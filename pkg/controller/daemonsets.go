package controller

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kubernetes-webhook-util/admission"
	hooks "github.com/appscode/kubernetes-webhook-util/admission/v1beta1"
	webhook "github.com/appscode/kubernetes-webhook-util/admission/v1beta1/workload"
	workload "github.com/appscode/kubernetes-webhook-util/workload/v1"
	ext_util "github.com/appscode/kutil/extensions/v1beta1"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (c *StashController) NewDaemonSetWebhook() hooks.AdmissionHook {
	return webhook.NewWorkloadWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "daemonsets",
		},
		"daemonset",
		"DaemonSet",
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				w := obj.(*workload.Workload)
				_, _, err := c.mutateDaemonSet(w)
				return w, err
			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				w := newObj.(*workload.Workload)
				_, _, err := c.mutateDaemonSet(w)
				return w, err
			},
		},
	)
}

func (c *StashController) initDaemonSetWatcher() {
	c.dsInformer = c.kubeInformerFactory.Extensions().V1beta1().DaemonSets().Informer()
	c.dsQueue = queue.New("DaemonSet", c.MaxNumRequeues, c.NumThreads, c.runDaemonSetInjector)
	c.dsInformer.AddEventHandler(queue.DefaultEventHandler(c.dsQueue.GetQueue()))
	c.dsLister = c.kubeInformerFactory.Extensions().V1beta1().DaemonSets().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the daemonset to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runDaemonSetInjector(key string) error {
	obj, exists, err := c.dsInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a DaemonSet, so that we will see a delete for one d
		glog.Warningf("DaemonSet %s does not exist anymore\n", key)
	} else {
		glog.Infof("Sync/Add/Update for DaemonSet %s\n", key)

		ds := obj.(*extensions.DaemonSet).DeepCopy()
		ds.GetObjectKind().SetGroupVersionKind(extensions.SchemeGroupVersion.WithKind(api.KindDaemonSet))

		w, err := workload.ConvertToWorkload(ds.DeepCopy())
		if err != nil {
			return nil
		}
		_, modified, err := c.mutateDaemonSet(w)
		if err != nil {
			return err
		}
		if modified {
			_, _, err := ext_util.PatchDaemonSetObject(c.kubeClient, ds, w.Object.(*extensions.DaemonSet))
			if err != nil {
				return err
			}
			return ext_util.WaitUntilDaemonSetReady(c.kubeClient, ds.ObjectMeta)
		}
	}
	return nil
}

func (c *StashController) mutateDaemonSet(w *workload.Workload) (*api.Restic, bool, error) {
	oldRestic, err := util.GetAppliedRestic(w.Annotations)
	if err != nil {
		return nil, false, err
	}

	newRestic, err := util.FindRestic(c.rstLister, w.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for DaemonSet %s/%s.", w.Name, w.Namespace)
		return nil, false, err
	}

	if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
		if !newRestic.Spec.Paused {
			err := c.ensureWorkloadSidecar(w, oldRestic, newRestic)
			if err != nil {
				return nil, false, err
			}
			workload.ApplyWorkload(w.Object, w)
			switch t := w.Object.(type) {
			case *extensions.DaemonSet:
				t.Spec.UpdateStrategy.Type = extensions.RollingUpdateDaemonSetStrategyType
				if t.Spec.UpdateStrategy.RollingUpdate == nil ||
					t.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable == nil ||
					t.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.IntValue() == 0 {
					count := intstr.FromInt(1)
					t.Spec.UpdateStrategy.RollingUpdate = &extensions.RollingUpdateDaemonSet{
						MaxUnavailable: &count,
					}
				}
			case *appsv1beta2.DaemonSet:
				t.Spec.UpdateStrategy.Type = appsv1beta2.RollingUpdateDaemonSetStrategyType
				if t.Spec.UpdateStrategy.RollingUpdate == nil ||
					t.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable == nil ||
					t.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.IntValue() == 0 {
					count := intstr.FromInt(1)
					t.Spec.UpdateStrategy.RollingUpdate = &appsv1beta2.RollingUpdateDaemonSet{
						MaxUnavailable: &count,
					}
				}
			case *appsv1.DaemonSet:
				t.Spec.UpdateStrategy.Type = appsv1.RollingUpdateDaemonSetStrategyType
				if t.Spec.UpdateStrategy.RollingUpdate == nil ||
					t.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable == nil ||
					t.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.IntValue() == 0 {
					count := intstr.FromInt(1)
					t.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateDaemonSet{
						MaxUnavailable: &count,
					}
				}
			}

			return newRestic, true, nil
		}
	} else if oldRestic != nil && newRestic == nil {
		err := c.ensureWorkloadSidecarDeleted(w, oldRestic)
		if err != nil {
			return nil, false, err
		}
		workload.ApplyWorkload(w.Object, w)
		return oldRestic, true, nil
	}
	return oldRestic, false, nil
}
