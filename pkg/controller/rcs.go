package controller

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kubernetes-webhook-util/admission"
	hooks "github.com/appscode/kubernetes-webhook-util/admission/v1beta1"
	webhook "github.com/appscode/kubernetes-webhook-util/admission/v1beta1/workload"
	wapi "github.com/appscode/kubernetes-webhook-util/apis/workload/v1"
	wcs "github.com/appscode/kubernetes-webhook-util/client/workload/v1"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

func (c *StashController) NewReplicationControllerWebhook() hooks.AdmissionHook {
	return webhook.NewWorkloadWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "replicationcontrollers",
		},
		"replicationcontroller",
		"ReplicationController",
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				w := obj.(*wapi.Workload)
				_, _, err := c.mutateReplicationController(w)
				return w, err

			},
			UpdateFunc: func(oldObj, newObj runtime.Object) (runtime.Object, error) {
				w := newObj.(*wapi.Workload)
				_, _, err := c.mutateReplicationController(w)
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
		err = util.DeleteConfigmapLock(c.kubeClient, ns, api.LocalTypedReference{Kind: api.KindReplicationController, Name: name})
		if err != nil && !kerr.IsNotFound(err) {
			return err
		}
	} else {
		glog.Infof("Sync/Add/Update for ReplicationController %s", key)

		rc := obj.(*core.ReplicationController).DeepCopy()
		rc.GetObjectKind().SetGroupVersionKind(core.SchemeGroupVersion.WithKind(api.KindReplicationController))

		w, err := wcs.ConvertToWorkload(rc.DeepCopy())
		if err != nil {
			return nil
		}
		restic, modified, err := c.mutateReplicationController(w)
		if err != nil {
			return err
		}
		if modified {
			_, _, err = core_util.PatchRCObject(c.kubeClient, rc, w.Object.(*core.ReplicationController))
			if err != nil {
				return err
			}
		}

		// ReplicationController does not have RollingUpdate strategy. We must delete old pods manually to get patched state.
		if restic != nil {
			err = c.forceRestartPods(w, restic)
			if err != nil {
				return err
			}
		}
		return core_util.WaitUntilRCReady(c.kubeClient, rc.ObjectMeta)
	}
	return nil
}

func (c *StashController) mutateReplicationController(w *wapi.Workload) (*api.Restic, bool, error) {
	oldRestic, err := util.GetAppliedRestic(w.Annotations)
	if err != nil {
		return nil, false, err
	}
	newRestic, err := util.FindRestic(c.rstLister, w.ObjectMeta)
	if err != nil {
		log.Errorf("Error while searching Restic for ReplicationController %s/%s.", w.Name, w.Namespace)
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

		err = util.DeleteConfigmapLock(c.kubeClient, w.Namespace, api.LocalTypedReference{Kind: api.KindReplicationController, Name: w.Name})
		if err != nil && !kerr.IsNotFound(err) {
			return nil, false, err
		}
		return oldRestic, true, nil
	}
	return oldRestic, false, nil
}
