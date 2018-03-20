package controller

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kutil/admission"
	hooks "github.com/appscode/kutil/admission/v1beta1"
	apps_util "github.com/appscode/kutil/apps/v1beta1"
	"github.com/appscode/kutil/tools/queue"
	workload "github.com/appscode/kutil/workload/v1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	//oneliner "github.com/the-redback/go-oneliners"
)

func (c *StashController) NewDeploymentWebhook() hooks.AdmissionHook {
	return hooks.NewWorkloadWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "deployments",
		},
		"deployment",
		appsv1beta1.SchemeGroupVersion.WithKind("Deployment"),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				modObj, _, err := c.mutateDeployment(obj.(*workload.Workload))
				return modObj, err

			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				modObj, _, err := c.mutateDeployment(newObj.(*workload.Workload))
				return modObj, err
			},
		},
	)
}

func (c *StashController) initDeploymentWatcher() {
	c.dpInformer = c.kubeInformerFactory.Apps().V1beta1().Deployments().Informer()
	c.dpQueue = queue.New("Deployment", c.MaxNumRequeues, c.NumThreads, c.runDeploymentInjector)
	c.dpInformer.AddEventHandler(queue.DefaultEventHandler(c.dpQueue.GetQueue()))
	c.dpLister = c.kubeInformerFactory.Apps().V1beta1().Deployments().Lister()
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
		util.DeleteConfigmapLock(c.KubeClient, ns, api.LocalTypedReference{Kind: api.KindDeployment, Name: name})
	} else {
		dp := obj.(*appsv1beta1.Deployment)
		glog.Infof("Sync/Add/Update for Deployment %s\n", key)

		w, err := workload.ConvertToWorkload(dp.DeepCopy())
		if err != nil {
			return nil
		}

		// mutateDeployment add or remove sidecar to Deployment when necessary
		modObj, modified, err := c.mutateDeployment(w)
		if err != nil {
			return err
		}

		if modified {
			patchedObj, _, err := apps_util.PatchDeployment(c.KubeClient, dp, func(obj *appsv1beta1.Deployment) *appsv1beta1.Deployment {
				return modObj.Object.(*appsv1beta1.Deployment)
			})
			if err != nil {
				return err
			}

			return apps_util.WaitUntilDeploymentReady(c.KubeClient, patchedObj.ObjectMeta)
		}
	}
	return nil
}

func (c *StashController) mutateDeployment(w *workload.Workload) (*workload.Workload, bool, error) {
	oldRestic, err := util.GetAppliedRestic(w.Annotations)
	if err != nil {
		return nil, false, err
	}

	newRestic, err := util.FindRestic(c.RstLister, w.ObjectMeta)
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
			workload.ApplyWorkload(w.Object, w)
			return w, true, nil
		}
	} else if oldRestic != nil && newRestic == nil {
		err := c.ensureWorkloadSidecarDeleted(w, oldRestic)
		if err != nil {
			return nil, false, err
		}
		workload.ApplyWorkload(w.Object, w)
		err = util.DeleteConfigmapLock(c.KubeClient, w.Namespace, api.LocalTypedReference{Kind: api.KindDeployment, Name: w.Name})
		if err != nil {
			return nil, false, err
		}
		return w, true, nil
	}

	return w, false, nil
}
