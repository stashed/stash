package controller

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kutil/admission"
	hooks "github.com/appscode/kutil/admission/v1beta1"
	ext_util "github.com/appscode/kutil/extensions/v1beta1"
	"github.com/appscode/kutil/tools/queue"
	workload "github.com/appscode/kutil/workload/v1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/apis/apps"
)

func (c *StashController) NewReplicaSetWebhook() hooks.AdmissionHook {
	return hooks.NewWorkloadWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "replicasets",
		},
		"replicaset",
		apps.SchemeGroupVersion.WithKind("ReplicaSet"),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				modObj, _, err := c.mutateReplicaSet(obj.(*workload.Workload))
				return modObj, err

			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				modObj, _, err := c.mutateReplicaSet(newObj.(*workload.Workload))
				return modObj, err
			},
		},
	)
}

func (c *StashController) initReplicaSetWatcher() {
	c.rsInformer = c.kubeInformerFactory.Extensions().V1beta1().ReplicaSets().Informer()
	c.rsQueue = queue.New("ReplicaSet", c.MaxNumRequeues, c.NumThreads, c.runReplicaSetInjector)
	c.rsInformer.AddEventHandler(queue.DefaultEventHandler(c.rsQueue.GetQueue()))
	c.rsLister = c.kubeInformerFactory.Extensions().V1beta1().ReplicaSets().Lister()
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
		util.DeleteConfigmapLock(c.KubeClient, ns, api.LocalTypedReference{Kind: api.KindReplicaSet, Name: name})
	} else {
		rs := obj.(*extensions.ReplicaSet)
		glog.Infof("Sync/Add/Update for ReplicaSet %s\n", key)

		if !ext_util.IsOwnedByDeployment(rs) {
			w, err := workload.ConvertToWorkload(rs.DeepCopy())
			if err != nil {
				return nil
			}

			mw, modified, err := c.mutateReplicaSet(w)
			if err != nil {
				return err
			}

			patchedObj := &extensions.ReplicaSet{}
			if modified {
				patchedObj, _, err = ext_util.PatchReplicaSet(c.KubeClient, rs, func(obj *extensions.ReplicaSet) *extensions.ReplicaSet {
					return mw.Object.(*extensions.ReplicaSet)
				})
				if err != nil {
					return err
				}
			}

			// ReplicaSet does not have RollingUpdate strategy. We must delete old pods manually to get patched state.
			if restartType := util.GetString(patchedObj.Annotations, util.ForceRestartType); restartType != "" {
				err := c.forceRestartRSPods(patchedObj, restartType, api.BackupType(util.GetString(patchedObj.Annotations, util.BackupType)))
				if err != nil {
					return err
				}
				return ext_util.WaitUntilReplicaSetReady(c.KubeClient, patchedObj.ObjectMeta)
			}
		}
	}
	return nil
}

func (c *StashController) mutateReplicaSet(w *workload.Workload) (*workload.Workload, bool, error) {
	oldRestic, err := util.GetAppliedRestic(w.Annotations)
	if err != nil {
		return nil, false, err
	}

	newRestic, err := util.FindRestic(c.RstLister, w.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for ReplicaSet %s/%s.", w.Name, w.Namespace)
		return nil, false, err
	}

	if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
		if !newRestic.Spec.Paused {
			err := c.ensureWorkloadSidecar(w, oldRestic, newRestic)
			if err != nil {
				return nil, false, err
			}
			w.Annotations[util.ForceRestartType] = util.SideCarRemoved
			w.Annotations[util.BackupType] = string(oldRestic.Spec.Type)
			workload.ApplyWorkload(w.Object, w)

			return w, true, nil
		}
	} else if oldRestic != nil && newRestic == nil {
		err := c.ensureWorkloadSidecarDeleted(w, oldRestic)
		if err != nil {
			return nil, false, err
		}
		w.Annotations[util.ForceRestartType] = util.SideCarRemoved
		w.Annotations[util.BackupType] = string(oldRestic.Spec.Type)
		workload.ApplyWorkload(w.Object, w)

		err = util.DeleteConfigmapLock(c.KubeClient, w.Namespace, api.LocalTypedReference{Kind: api.KindReplicaSet, Name: w.Name})
		if err != nil {
			return nil, false, err
		}
		return w, true, nil
	}
	return w, false, nil
}

func (c *StashController) forceRestartRSPods(rs *extensions.ReplicaSet, restartType string, backupType api.BackupType) error {
	rs, _, err := ext_util.PatchReplicaSet(c.KubeClient, rs, func(obj *extensions.ReplicaSet) *extensions.ReplicaSet {
		delete(obj.Annotations, util.ForceRestartType)
		delete(obj.Annotations, util.BackupType)
		return obj
	})
	if err != nil {
		return err
	}

	if restartType == util.SideCarAdded {
		err := util.WaitUntilSidecarAdded(c.KubeClient, rs.Namespace, rs.Spec.Selector, backupType)
		if err != nil {
			return err
		}
	} else if restartType == util.SideCarRemoved {
		err := util.WaitUntilSidecarRemoved(c.KubeClient, rs.Namespace, rs.Spec.Selector, backupType)
		if err != nil {
			return err
		}
	}
	return nil
}
