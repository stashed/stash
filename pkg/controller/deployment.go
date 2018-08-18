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
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

func (c *StashController) NewDeploymentWebhook() hooks.AdmissionHook {
	return webhook.NewWorkloadWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "deployments",
		},
		"deployment",
		"Deployment",
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				w := obj.(*wapi.Workload)
				_, _, err := c.mutateDeployment(w)
				return w, err

			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				w := newObj.(*wapi.Workload)
				_, _, err := c.mutateDeployment(w)
				return w, err
			},
		},
	)
}

func (c *StashController) initDeploymentWatcher() {
	c.dpInformer = c.kubeInformerFactory.Apps().V1().Deployments().Informer()
	c.dpQueue = queue.New("Deployment", c.MaxNumRequeues, c.NumThreads, c.runDeploymentInjector)
	c.dpInformer.AddEventHandler(queue.DefaultEventHandler(c.dpQueue.GetQueue()))
	c.dpLister = c.kubeInformerFactory.Apps().V1().Deployments().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runDeploymentInjector(key string) error {
	obj, exists, err := c.dpInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Deployment, so that we will see a delete for one d
		glog.Warningf("Deployment %s does not exist anymore\n", key)

		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		err = util.DeleteConfigmapLock(c.kubeClient, ns, api.LocalTypedReference{Kind: api.KindDeployment, Name: name})
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
	} else {
		glog.Infof("Sync/Add/Update for Deployment %s", key)

		dp := obj.(*appsv1.Deployment).DeepCopy()
		dp.GetObjectKind().SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind(api.KindDeployment))

		w, err := wcs.ConvertToWorkload(dp.DeepCopy())
		if err != nil {
			return nil
		}
		// mutateDeployment add or remove sidecar to Deployment when necessary
		_, modified, err := c.mutateDeployment(w)
		if err != nil {
			return err
		}
		if modified {
			_, _, err := apps_util.PatchDeploymentObject(c.kubeClient, dp, w.Object.(*appsv1.Deployment))
			if err != nil {
				return err
			}
			return apps_util.WaitUntilDeploymentReady(c.kubeClient, dp.ObjectMeta)
		}
	}
	return nil
}

func (c *StashController) mutateDeployment(w *wapi.Workload) (*api.Restic, bool, error) {
	oldRestic, err := util.GetAppliedRestic(w.Annotations)
	if err != nil {
		return nil, false, err
	}

	newRestic, err := util.FindRestic(c.rstLister, w.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for Deployment %s/%s.", w.Name, w.Namespace)
		return nil, false, err
	}

	if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
		if !newRestic.Spec.Paused {
			err := c.ensureWorkloadSidecar(w, oldRestic, newRestic)
			if err != nil {
				return nil, false, err
			}
			wcs.ApplyWorkload(w.Object, w)
			return newRestic, true, nil
		}
	} else if oldRestic != nil && newRestic == nil {
		err := c.ensureWorkloadSidecarDeleted(w, oldRestic)
		if err != nil {
			return nil, false, err
		}
		wcs.ApplyWorkload(w.Object, w)
		err = util.DeleteConfigmapLock(c.kubeClient, w.Namespace, api.LocalTypedReference{Kind: api.KindDeployment, Name: w.Name})
		if err != nil && !kerr.IsNotFound(err) {
			return nil, false, err
		}
		return oldRestic, true, nil
	}

	return oldRestic, false, nil
}
