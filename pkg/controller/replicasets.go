package controller

import (
	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	apps "k8s.io/api/apps/v1"
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

func (c *StashController) NewReplicaSetWebhook() hooks.AdmissionHook {
	return webhook.NewWorkloadWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "replicasets",
		},
		"replicaset",
		"ReplicaSet",
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				w := obj.(*wapi.Workload)
				// if ReplicaSet is owned by a Deployment, don't process it.
				if !apps_util.IsOwnedByDeployment(w.OwnerReferences) {
					// apply stash backup/restore logic on this workload
					_, err := c.applyStashLogic(w)
					return w, err
				}
				return w, nil
			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				w := newObj.(*wapi.Workload)
				// if ReplicaSet is owned by a Deployment, don't process it.
				if !apps_util.IsOwnedByDeployment(w.OwnerReferences) {
					// apply stash backup/restore logic on this workload
					_, err := c.applyStashLogic(w)
					return w, err
				}
				return w, nil
			},
		},
	)
}

func (c *StashController) initReplicaSetWatcher() {
	c.rsInformer = c.kubeInformerFactory.Apps().V1().ReplicaSets().Informer()
	c.rsQueue = queue.New("ReplicaSet", c.MaxNumRequeues, c.NumThreads, c.runReplicaSetInjector)
	c.rsInformer.AddEventHandler(queue.DefaultEventHandler(c.rsQueue.GetQueue()))
	c.rsLister = c.kubeInformerFactory.Apps().V1().ReplicaSets().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runReplicaSetInjector(key string) error {
	obj, exists, err := c.rsInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a ReplicaSet, so that we will see a delete for one d
		glog.Warningf("ReplicaSet %s does not exist anymore\n", key)

		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		// workload does not exist anymore. so delete respective ConfigMapLocks if exist
		err = util.DeleteAllConfigMapLocks(c.kubeClient, ns, name, apis.KindReplicaSet)
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
	} else {
		glog.Infof("Sync/Add/Update for ReplicaSet %s", key)

		rs := obj.(*apps.ReplicaSet).DeepCopy()
		rs.GetObjectKind().SetGroupVersionKind(apps.SchemeGroupVersion.WithKind(apis.KindReplicaSet))

		// convert ReplicaSet into a common object (Workload type) so that
		// we don't need to re-write stash logic for ReplicaSet separately
		w, err := wcs.ConvertToWorkload(rs.DeepCopy())
		if err != nil {
			glog.Errorf("failed to convert ReplicaSet %s/%s to workload type. Reason: %v", rs.Namespace, rs.Name, err)
			return err
		}

		// if ReplicaSet is owned by a Deployment, don't process it.
		if !apps_util.IsOwnedByDeployment(w.OwnerReferences) {
			// apply stash backup/restore logic on this workload
			modified, err := c.applyStashLogic(w)
			if err != nil {
				return err
			}

			if modified {
				// workload has been modified. patch the workload so that respective pods start with the updated spec
				_, _, err = apps_util.PatchReplicaSetObject(c.kubeClient, rs, w.Object.(*apps.ReplicaSet))
				if err != nil {
					glog.Errorf("failed to update ReplicaSet %s/%s. Reason: %v", rs.Namespace, rs.Name, err)
					return err
				}
			}

			// ReplicaSet does not have RollingUpdate strategy. We must delete old pods manually to get patched state.
			stateChanged, err := c.ensureWorkloadLatestState(w)
			if err != nil {
				return err
			}

			if stateChanged {
				// wait until newly patched ReplicaSet pods are ready
				err = util.WaitUntilReplicaSetReady(c.kubeClient, rs.ObjectMeta)
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

	}
	return nil
}
