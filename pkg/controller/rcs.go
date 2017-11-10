package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	stringz "github.com/appscode/go/strings"
	"github.com/appscode/kutil"
	core_util "github.com/appscode/kutil/core/v1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	core_listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (c *StashController) initRCWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			return c.k8sClient.CoreV1().ReplicationControllers(core.NamespaceAll).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.k8sClient.CoreV1().ReplicationControllers(core.NamespaceAll).Watch(options)
		},
	}

	// create the workqueue
	c.rcQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rc")

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pod key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the ReplicationController than the version which was responsible for triggering the update.
	c.rcIndexer, c.rcInformer = cache.NewIndexerInformer(lw, &core.ReplicationController{}, c.options.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.rcQueue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				c.rcQueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.rcQueue.Add(key)
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	c.rcLister = core_listers.NewReplicationControllerLister(c.rcIndexer)
}

func (c *StashController) runRCWatcher() {
	for c.processNextRC() {
	}
}

func (c *StashController) processNextRC() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.rcQueue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two deployments with the same key are never processed in
	// parallel.
	defer c.rcQueue.Done(key)

	// Invoke the method containing the business logic
	err := c.runRCInjector(key.(string))
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.rcQueue.Forget(key)
		return true
	}
	log.Errorf("Failed to process ReplicationController %v. Reason: %s", key, err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.rcQueue.NumRequeues(key) < c.options.MaxNumRequeues {
		glog.Infof("Error syncing deployment %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.rcQueue.AddRateLimited(key)
		return true
	}

	c.rcQueue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.Infof("Dropping deployment %q out of the queue: %v", key, err)
	return true
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runRCInjector(key string) error {
	obj, exists, err := c.rcIndexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a ReplicationController, so that we will see a delete for one d
		fmt.Printf("ReplicationController %s does not exist anymore\n", key)
	} else {
		rc := obj.(*core.ReplicationController)
		fmt.Printf("Sync/Add/Update for ReplicationController %s\n", rc.GetName())

		oldRestic, err := util.GetAppliedRestic(rc.Annotations)
		if err != nil {
			return err
		}
		newRestic, err := util.FindRestic(c.rstLister, rc.ObjectMeta)
		if err != nil {
			log.Errorf("Error while searching Restic for ReplicationController %s/%s.", rc.Name, rc.Namespace)
			return err
		}
		if util.ResticEqual(oldRestic, newRestic) {
			return nil
		}
		if newRestic != nil {
			return c.EnsureReplicationControllerSidecar(rc, oldRestic, newRestic)
		} else if oldRestic != nil {
			return c.EnsureReplicationControllerSidecarDeleted(rc, oldRestic)
		}
	}
	return nil
}

func (c *StashController) EnsureReplicationControllerSidecar(resource *core.ReplicationController, old, new *api.Restic) (err error) {
	if new.Spec.Backend.StorageSecretName == "" {
		err = fmt.Errorf("missing repository secret name for Restic %s/%s", new.Namespace, new.Name)
		return
	}
	_, err = c.k8sClient.CoreV1().Secrets(resource.Namespace).Get(new.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if c.options.EnableRBAC {
		sa := stringz.Val(resource.Spec.Template.Spec.ServiceAccountName, "default")
		err := c.ensureRoleBinding(kutil.GetObjectReference(resource, core.SchemeGroupVersion), sa)
		if err != nil {
			return err
		}
	}

	resource, err = core_util.PatchRC(c.k8sClient, resource, func(obj *core.ReplicationController) *core.ReplicationController {
		workload := api.LocalTypedReference{
			Kind: api.AppKindReplicationController,
			Name: obj.Name,
		}
		obj.Spec.Template.Spec.Containers = core_util.UpsertContainer(obj.Spec.Template.Spec.Containers, util.CreateSidecarContainer(new, c.options.SidecarImageTag, workload))
		obj.Spec.Template.Spec.Volumes = util.UpsertScratchVolume(obj.Spec.Template.Spec.Volumes)
		obj.Spec.Template.Spec.Volumes = util.UpsertDownwardVolume(obj.Spec.Template.Spec.Volumes)
		obj.Spec.Template.Spec.Volumes = util.MergeLocalVolume(obj.Spec.Template.Spec.Volumes, old, new)

		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}
		r := &api.Restic{
			TypeMeta: metav1.TypeMeta{
				APIVersion: api.SchemeGroupVersion.String(),
				Kind:       api.ResourceKindRestic,
			},
			ObjectMeta: new.ObjectMeta,
			Spec:       new.Spec,
		}
		data, _ := kutil.MarshalToJson(r, api.SchemeGroupVersion)
		obj.Annotations[api.LastAppliedConfiguration] = string(data)
		obj.Annotations[api.VersionTag] = c.options.SidecarImageTag

		return obj
	})
	if err != nil {
		return
	}

	err = core_util.WaitUntilRCReady(c.k8sClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	err = util.WaitUntilSidecarAdded(c.k8sClient, resource.Namespace, &metav1.LabelSelector{MatchLabels: resource.Spec.Selector})
	return err
}

func (c *StashController) EnsureReplicationControllerSidecarDeleted(resource *core.ReplicationController, restic *api.Restic) (err error) {
	if c.options.EnableRBAC {
		err := c.ensureRoleBindingDeleted(resource.ObjectMeta)
		if err != nil {
			return err
		}
	}

	resource, err = core_util.PatchRC(c.k8sClient, resource, func(obj *core.ReplicationController) *core.ReplicationController {
		obj.Spec.Template.Spec.Containers = core_util.EnsureContainerDeleted(obj.Spec.Template.Spec.Containers, util.StashContainer)
		obj.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(obj.Spec.Template.Spec.Volumes, util.ScratchDirVolumeName)
		obj.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(obj.Spec.Template.Spec.Volumes, util.PodinfoVolumeName)
		if restic.Spec.Backend.Local != nil {
			obj.Spec.Template.Spec.Volumes = util.EnsureVolumeDeleted(obj.Spec.Template.Spec.Volumes, util.LocalVolumeName)
		}
		if obj.Annotations != nil {
			delete(obj.Annotations, api.LastAppliedConfiguration)
			delete(obj.Annotations, api.VersionTag)
		}
		return obj
	})
	if err != nil {
		return
	}

	err = core_util.WaitUntilRCReady(c.k8sClient, resource.ObjectMeta)
	if err != nil {
		return
	}
	err = util.WaitUntilSidecarRemoved(c.k8sClient, resource.Namespace, &metav1.LabelSelector{MatchLabels: resource.Spec.Selector})
	return err
}
