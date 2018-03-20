package controller

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kutil/admission"
	hooks "github.com/appscode/kutil/admission/v1beta1"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/appscode/kutil/tools/queue"
	workload "github.com/appscode/kutil/workload/v1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

func (c *StashController) NewReplicationControllerWebhook() hooks.AdmissionHook {
	return hooks.NewWorkloadWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "replicationcontrollers",
		},
		"replicationcontroller",
		core.SchemeGroupVersion.WithKind("ReplicationController"),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				modObj, _, err := c.mutateReplicationController(obj.(*workload.Workload))
				return modObj, err

			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				modObj, _, err := c.mutateReplicationController(newObj.(*workload.Workload))
				return modObj, err
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
		util.DeleteConfigmapLock(c.KubeClient, ns, api.LocalTypedReference{Kind: api.KindReplicationController, Name: name})
	} else {
		rc := obj.(*core.ReplicationController)
		glog.Infof("Sync/Add/Update for ReplicationController %s\n", key)

		w, err := workload.ConvertToWorkload(rc.DeepCopy())
		if err != nil {
			return nil
		}

		modObj, modified, err := c.mutateReplicationController(w)
		if err != nil {
			return err
		}

		patchedObj := &core.ReplicationController{}
		if modified {
			patchedObj, _, err = core_util.PatchRC(c.KubeClient, rc, func(obj *core.ReplicationController) *core.ReplicationController {
				return modObj.Object.(*core.ReplicationController)
			})
			if err != nil {
				return err
			}
		}

		// ReplicationController does not have RollingUpdate strategy. We must delete old pods manually to get patched state.
		if restartType := util.GetString(patchedObj.Annotations, util.ForceRestartType); restartType != "" {
			err := c.forceRestartRCPods(patchedObj, restartType, api.BackupType(util.GetString(patchedObj.Annotations, util.BackupType)))
			if err != nil {
				return err
			}
			return core_util.WaitUntilRCReady(c.KubeClient, patchedObj.ObjectMeta)
		}
	}
	return nil
}

func (c *StashController) mutateReplicationController(w *workload.Workload) (*workload.Workload, bool, error) {
	oldRestic, err := util.GetAppliedRestic(w.Annotations)
	if err != nil {
		return nil, false, err
	}
	newRestic, err := util.FindRestic(c.RstLister, w.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for ReplicationController %s/%s.", w.Name, w.Namespace)
		return nil, false, err
	}

	if newRestic != nil && !util.ResticEqual(oldRestic, newRestic) {
		if !newRestic.Spec.Paused {
			err := c.ensureWorkloadSidecar(w,api.KindReplicationController ,oldRestic, newRestic)
			if err != nil {
				return nil, false, err
			}
			w.Annotations[util.ForceRestartType] = util.SideCarAdded
			w.Annotations[util.BackupType] = string(newRestic.Spec.Type)
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

		err = util.DeleteConfigmapLock(c.KubeClient, w.Namespace, api.LocalTypedReference{Kind: api.KindReplicationController, Name: w.Name})
		if err != nil {
			return nil, false, err
		}
		return w, true, nil
	}
	return w, false, nil
}

func (c *StashController) forceRestartRCPods(rc *core.ReplicationController, restartType string, backupType api.BackupType) error {
	rc, _, err := core_util.PatchRC(c.KubeClient, rc, func(obj *core.ReplicationController) *core.ReplicationController {
		delete(obj.Annotations, util.ForceRestartType)
		delete(obj.Annotations, util.BackupType)
		return obj
	})
	if err != nil {
		return err
	}

	if restartType == util.SideCarAdded {
		err := util.WaitUntilSidecarAdded(c.KubeClient, rc.Namespace, &metav1.LabelSelector{MatchLabels: rc.Spec.Selector}, backupType)
		if err != nil {
			return err
		}
	} else if restartType == util.SideCarRemoved {
		err := util.WaitUntilSidecarRemoved(c.KubeClient, rc.Namespace, &metav1.LabelSelector{MatchLabels: rc.Spec.Selector}, backupType)
		if err != nil {
			return err
		}
	}
	return nil
}
