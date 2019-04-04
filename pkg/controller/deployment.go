package controller

import (
	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	apps_util "kmodules.xyz/client-go/apps/v1"
	"kmodules.xyz/client-go/tools/queue"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/workload"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

func (c *StashController) NewDeploymentWebhook() hooks.AdmissionHook {
	return webhook.NewWorkloadWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "deploymentmutators",
		},
		"deploymentmutator",
		"DeploymentMutator",
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				w := obj.(*wapi.Workload)
				// apply stash backup/restore logic on this workload
				_, err := c.applyStashLogic(w)
				return w, err

			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				w := newObj.(*wapi.Workload)
				// apply stash backup/restore logic on this workload
				_, err := c.applyStashLogic(w)
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
		// Below we will warm up our cache with a Deployment, so that we will see a delete for one deployment
		glog.Warningf("Deployment %s does not exist anymore\n", key)

		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		// workload does not exist anymore. so delete respective ConfigMapLocks if exist
		err = util.DeleteAllConfigMapLocks(c.kubeClient, ns, name, apis.KindDeployment)
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
	} else {
		glog.Infof("Sync/Add/Update for Deployment %s", key)

		dp := obj.(*appsv1.Deployment).DeepCopy()
		dp.GetObjectKind().SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind(apis.KindDeployment))

		// convert Deployment into a common object (Workload type) so that
		// we don't need to re-write stash logic for Deployment separately
		w, err := wcs.ConvertToWorkload(dp.DeepCopy())
		if err != nil {
			glog.Errorf("failed to convert Deployment %s/%s to workload type. Reason: %v", dp.Namespace, dp.Name, err)
			return err
		}

		// apply stash backup/restore logic on this workload
		modified, err := c.applyStashLogic(w)
		if err != nil {
			glog.Errorf("failed to apply stash logic on Deployment %s/%s. Reason: %v", dp.Namespace, dp.Name, err)
			return err
		}

		if modified {
			// workload has been modified. Patch the workload so that respective pods start with the updated spec
			_, _, err := apps_util.PatchDeploymentObject(c.kubeClient, dp, w.Object.(*appsv1.Deployment))
			if err != nil {
				glog.Errorf("failed to update Deployment %s/%s. Reason: %v", dp.Namespace, dp.Name, err)
				return err
			}

			//TODO: Should we force restart all pods while restore?
			// otherwise one pod will restore while others are writing/reading?

			// wait until newly patched deployment pods are ready
			err = util.WaitUntilDeploymentReady(c.kubeClient, dp.ObjectMeta)
			if err != nil {
				return err
			}
		}

		// if the workload does not have any stash sidecar/init-container then
		// delete respective ConfigMapLock and RBAC stuffs if exist
		err = c.ensureUnnecessaryConfigMapLockDeleted(w)
		if err != nil {
			return err
		}
		err = c.ensureUnnecessaryWorkloadRBACDeleted(w)
		if err != nil {
			return err
		}
	}
	return nil
}
