package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/appscode/kutil"
	ext_util "github.com/appscode/kutil/extensions/v1beta1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	stash_listers "github.com/appscode/stash/listers/stash/v1alpha1"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (c *StashController) initResticWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			return c.stashClient.Restics(apiv1.NamespaceAll).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.stashClient.Restics(apiv1.NamespaceAll).Watch(options)
		},
	}

	// create the workqueue
	c.rQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "restic")

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pod key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Restic than the version which was responsible for triggering the update.
	c.rIndexer, c.rInformer = cache.NewIndexerInformer(lw, &api.Restic{}, c.options.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.rQueue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			oldObj, ok := old.(*api.Restic)
			if !ok {
				log.Errorln("Invalid Restic object")
				return
			}
			newObj, ok := new.(*api.Restic)
			if !ok {
				log.Errorln("Invalid Restic object")
				return
			}
			if !util.ResticEqual(oldObj, newObj) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err == nil {
					c.rQueue.Add(key)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.rQueue.Add(key)
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	c.rLister = stash_listers.NewResticLister(c.rIndexer)
}

func (c *StashController) runResticWatcher() {
	for c.processNextRestic() {
	}
}

func (c *StashController) processNextRestic() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.rQueue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two deployments with the same key are never processed in
	// parallel.
	defer c.rQueue.Done(key)

	// Invoke the method containing the business logic
	err := c.runResticInjector(key.(string))
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.rQueue.Forget(key)
		return true
	}
	log.Errorf("Failed to process Restic %v. Reason: %s", key, err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.rQueue.NumRequeues(key) < c.options.MaxNumRequeues {
		glog.Infof("Error syncing deployment %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.rQueue.AddRateLimited(key)
		return true
	}

	c.rQueue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.Infof("Dropping deployment %q out of the queue: %v", key, err)
	return true
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runResticInjector(key string) error {
	obj, exists, err := c.rIndexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Restic, so that we will see a delete for one d
		fmt.Printf("Restic %s does not exist anymore\n", key)

		namespace, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		c.EnsureSidecarDeleted(namespace, name)
	} else {
		d := obj.(*api.Restic)
		fmt.Printf("Sync/Add/Update for Restic %s\n", d.GetName())

		c.EnsureSidecar(d)
		c.EnsureSidecarDeleted(d.Namespace, d.Name)
	}
	return nil
}

func (c *StashController) EnsureSidecar(restic *api.Restic) {
	sel, err := metav1.LabelSelectorAsSelector(&restic.Spec.Selector)
	if err != nil {
		c.recorder.Eventf(
			restic.ObjectReference(),
			apiv1.EventTypeWarning,
			eventer.EventReasonInvalidRestic,
			"Reason: %s",
			err.Error(),
		)
		return
	}
	{
		if resources, err := c.dpLister.Deployments(restic.Namespace).List(sel); err == nil {
			for _, resource := range resources {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.dpQueue.Add(key)
				}
			}
		}
	}
	{
		if resources, err := c.dsLister.DaemonSets(restic.Namespace).List(sel); err == nil {
			for _, resource := range resources {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.dsQueue.Add(key)
				}
			}
		}
	}
	//{
	//	if resources, err := c.ssLister.StatefulSets(restic.Namespace).List(sel); err == nil {
	//		for _, resource := range resources {
	//			key, err := cache.MetaNamespaceKeyFunc(resource)
	//			if err == nil {
	//				c.ssQueue.Add(key)
	//			}
	//		}
	//	}
	//}
	{
		if resources, err := c.rcLister.ReplicationControllers(restic.Namespace).List(sel); err == nil {
			for _, resource := range resources {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.rcQueue.Add(key)
				}
			}
		}
	}
	{
		if resources, err := c.rsLister.ReplicaSets(restic.Namespace).List(sel); err == nil {
			for _, resource := range resources {
				// If owned by a Deployment, skip it.
				if ext_util.IsOwnedByDeployment(resource) {
					continue
				}
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.rsQueue.Add(key)
				}
			}
		}
	}
}

func (c *StashController) EnsureSidecarDeleted(namespace, name string) {
	if resources, err := c.dpLister.Deployments(namespace).List(labels.Everything()); err == nil {
		for _, resource := range resources {
			restic, err := util.GetAppliedRestic(resource.Annotations)
			if err != nil {
				c.recorder.Eventf(
					kutil.GetObjectReference(resource, apps.SchemeGroupVersion),
					apiv1.EventTypeWarning,
					eventer.EventReasonInvalidRestic,
					"Reason: %s",
					err.Error(),
				)
			} else if restic != nil && restic.Namespace == namespace && restic.Name == name {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.dpQueue.Add(key)
				}
			}
		}
	}
	if resources, err := c.dsLister.DaemonSets(namespace).List(labels.Everything()); err == nil {
		for _, resource := range resources {
			restic, err := util.GetAppliedRestic(resource.Annotations)
			if err != nil {
				c.recorder.Eventf(
					kutil.GetObjectReference(resource, extensions.SchemeGroupVersion),
					apiv1.EventTypeWarning,
					eventer.EventReasonInvalidRestic,
					"Reason: %s",
					err.Error(),
				)
			} else if restic != nil && restic.Namespace == namespace && restic.Name == name {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.dsQueue.Add(key)
				}
			}
		}
	}
	//if resources, err := c.ssLister.StatefulSets(namespace).List(labels.Everything()); err == nil {
	//	for _, resource := range resources {
	//		restic, err := util.GetAppliedRestic(resource.Annotations)
	//		if err != nil {
	//			c.recorder.Eventf(
	//				kutil.GetObjectReference(resource, apps.SchemeGroupVersion),
	//				apiv1.EventTypeWarning,
	//				eventer.EventReasonInvalidRestic,
	//				"Reason: %s",
	//				err.Error(),
	//			)
	//		} else if restic != nil && restic.Namespace == namespace && restic.Name == name {
	//			key, err := cache.MetaNamespaceKeyFunc(resource)
	//			if err == nil {
	//				c.ssQueue.Add(key)
	//			}
	//		}
	//	}
	//}
	if resources, err := c.rcLister.ReplicationControllers(namespace).List(labels.Everything()); err == nil {
		for _, resource := range resources {
			restic, err := util.GetAppliedRestic(resource.Annotations)
			if err != nil {
				c.recorder.Eventf(
					kutil.GetObjectReference(resource, apiv1.SchemeGroupVersion),
					apiv1.EventTypeWarning,
					eventer.EventReasonInvalidRestic,
					"Reason: %s",
					err.Error(),
				)
			} else if restic != nil && restic.Namespace == namespace && restic.Name == name {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.rcQueue.Add(key)
				}
			}
		}
	}
	if resources, err := c.rsLister.ReplicaSets(namespace).List(labels.Everything()); err == nil {
		for _, resource := range resources {
			restic, err := util.GetAppliedRestic(resource.Annotations)
			if err != nil {
				c.recorder.Eventf(
					kutil.GetObjectReference(resource, extensions.SchemeGroupVersion),
					apiv1.EventTypeWarning,
					eventer.EventReasonInvalidRestic,
					"Reason: %s",
					err.Error(),
				)
			} else if restic != nil && restic.Namespace == namespace && restic.Name == name {
				key, err := cache.MetaNamespaceKeyFunc(resource)
				if err == nil {
					c.rsQueue.Add(key)
				}
			}
		}
	}
}
