package controller

import (
	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/tools/queue"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/workload"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
	wcs "kmodules.xyz/webhook-runtime/client/workload/v1"
)

func (c *StashController) NewReplicationControllerWebhook() hooks.AdmissionHook {
	return webhook.NewWorkloadWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "replicationcontrollermutators",
		},
		"replicationcontrollermutator",
		"ReplicationControllerMutator",
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

func (c *StashController) initRCWatcher() {
	c.rcInformer = c.kubeInformerFactory.Core().V1().ReplicationControllers().Informer()
	c.rcQueue = queue.New("ReplicationController", c.MaxNumRequeues, c.NumThreads, c.runRCInjector)
	c.rcInformer.AddEventHandler(queue.DefaultEventHandler(c.rcQueue.GetQueue()))
	c.rcLister = c.kubeInformerFactory.Core().V1().ReplicationControllers().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runRCInjector(key string) error {
	obj, exists, err := c.rcInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a ReplicationController, so that we will see a delete for one d
		glog.Warningf("ReplicationController %s does not exist anymore\n", key)

		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		// workload does not exist anymore. so delete respective ConfigMapLocks if exist
		err = util.DeleteAllConfigMapLocks(c.kubeClient, ns, name, apis.KindReplicationController)
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
	} else {
		glog.Infof("Sync/Add/Update for ReplicationController %s", key)

		rc := obj.(*core.ReplicationController).DeepCopy()
		rc.GetObjectKind().SetGroupVersionKind(core.SchemeGroupVersion.WithKind(apis.KindReplicationController))

		// convert ReplicationController into a common object (Workload type) so that
		// we don't need to re-write stash logic for ReplicationController separately
		w, err := wcs.ConvertToWorkload(rc.DeepCopy())
		if err != nil {
			glog.Errorf("failed to convert replicationcontroller %s/%s to workload type. Reason: %v", rc.Namespace, rc.Name, err)
			return err
		}

		// apply stash backup/restore logic on this workload
		modified, err := c.applyStashLogic(w)
		if err != nil {
			glog.Errorf("failed to apply stash logic on replicationcontroller %s/%s. Reason: %v", rc.Namespace, rc.Name, err)
			return err
		}

		if modified {
			// workload has been modified. patch the workload so that respective pods start with the updated spec
			_, _, err = core_util.PatchRCObject(c.kubeClient, rc, w.Object.(*core.ReplicationController))
			if err != nil {
				glog.Errorf("failed to update replicationcontroller %s/%s. Reason: %v", rc.Namespace, rc.Name, err)
				return err
			}
		}

		// ReplicationController does not have RollingUpdate strategy. We must delete old pods manually to get patched state.
		stateChanged, err := c.ensureWorkloadLatestState(w)
		if err != nil {
			return err
		}

		if stateChanged {
			// wait until newly patched RC pods are ready
			err = util.WaitUntilRCReady(c.kubeClient, rc.ObjectMeta)
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
