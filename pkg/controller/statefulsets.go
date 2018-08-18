package controller

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kubernetes-webhook-util/admission"
	hooks "github.com/appscode/kubernetes-webhook-util/admission/v1beta1"
	webhook "github.com/appscode/kubernetes-webhook-util/admission/v1beta1/workload"
	wapi "github.com/appscode/kubernetes-webhook-util/apis/workload/v1"
	wcs "github.com/appscode/kubernetes-webhook-util/client/workload/v1"
	apps_util "github.com/appscode/kutil/apps/v1"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (c *StashController) NewStatefulSetWebhook() hooks.AdmissionHook {
	return webhook.NewWorkloadWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "statefulsets",
		},
		"statefulset",
		"StatefulSet",
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				w := obj.(*wapi.Workload)
				_, _, err := c.mutateStatefulSet(w)
				return w, err

			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				w := newObj.(*wapi.Workload)
				_, _, err := c.mutateStatefulSet(w)
				return w, err
			},
		},
	)
}

func (c *StashController) initStatefulSetWatcher() {
	c.ssInformer = c.kubeInformerFactory.Apps().V1().StatefulSets().Informer()
	c.ssQueue = queue.New("StatefulSet", c.MaxNumRequeues, c.NumThreads, c.runStatefulSetInjector)
	c.ssInformer.AddEventHandler(queue.DefaultEventHandler(c.ssQueue.GetQueue()))
	c.ssLister = c.kubeInformerFactory.Apps().V1().StatefulSets().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runStatefulSetInjector(key string) error {
	obj, exists, err := c.ssInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a StatefulSet, so that we will see a delete for one d
		glog.Warningf("StatefulSet %s does not exist anymore\n", key)
	} else {
		glog.Infof("Sync/Add/Update for StatefulSet %s", key)

		ss := obj.(*appsv1.StatefulSet).DeepCopy()
		ss.GetObjectKind().SetGroupVersionKind(appsv1beta1.SchemeGroupVersion.WithKind(api.KindStatefulSet))

		w, err := wcs.ConvertToWorkload(ss.DeepCopy())
		if err != nil {
			return nil
		}
		_, modified, err := c.mutateStatefulSet(w)
		if err != nil {
			return nil
		}
		if modified {
			_, _, err := apps_util.PatchStatefulSetObject(c.kubeClient, ss, w.Object.(*appsv1.StatefulSet))
			if err != nil {
				return err
			}

			return apps_util.WaitUntilStatefulSetReady(c.kubeClient, ss.ObjectMeta)
		}
	}
	return nil
}

func (c *StashController) mutateStatefulSet(w *wapi.Workload) (*api.Restic, bool, error) {
	oldRestic, err := util.GetAppliedRestic(w.Annotations)
	if err != nil {
		return nil, false, err
	}

	newRestic, err := util.FindRestic(c.rstLister, w.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for StatefulSet %s/%s.", w.Name, w.Namespace)
		return nil, false, err
	}

	if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
		if !newRestic.Spec.Paused {
			err := c.ensureWorkloadSidecar(w, oldRestic, newRestic)
			if err != nil {
				return nil, false, err
			}
			wcs.ApplyWorkload(w.Object, w)
			switch t := w.Object.(type) {
			case *appsv1beta1.StatefulSet:
				t.Spec.UpdateStrategy.Type = appsv1beta1.RollingUpdateStatefulSetStrategyType
			case *appsv1beta2.StatefulSet:
				t.Spec.UpdateStrategy.Type = appsv1beta2.RollingUpdateStatefulSetStrategyType
			case *appsv1.StatefulSet:
				t.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
			}
			return newRestic, true, nil
		}
	} else if oldRestic != nil && newRestic == nil {
		err := c.ensureWorkloadSidecarDeleted(w, oldRestic)
		if err != nil {
			return nil, false, err
		}
		wcs.ApplyWorkload(w.Object, w)
		switch t := w.Object.(type) {
		case *appsv1beta1.StatefulSet:
			t.Spec.UpdateStrategy.Type = appsv1beta1.RollingUpdateStatefulSetStrategyType
		case *appsv1beta2.StatefulSet:
			t.Spec.UpdateStrategy.Type = appsv1beta2.RollingUpdateStatefulSetStrategyType
		case *appsv1.StatefulSet:
			t.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
		}
		return oldRestic, true, nil
	}
	return oldRestic, false, nil
}
