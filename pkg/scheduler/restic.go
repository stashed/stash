package scheduler

import (
	"fmt"

	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	stash_listers "github.com/appscode/stash/listers/stash/v1alpha1"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (c *Scheduler) initResticWatcher() {
	// TODO: Watch one Restic object, when support for Kubernetes 1.8 is dropped.
	// ref: https://github.com/kubernetes/kubernetes/pull/53345
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			return c.stashClient.Restics(c.opt.Namespace).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.stashClient.Restics(c.opt.Namespace).Watch(options)
		},
	}

	// create the workqueue
	c.rQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "restic")

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pod key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Restic than the version which was responsible for triggering the update.
	c.rIndexer, c.rInformer = cache.NewIndexerInformer(lw, &api.Restic{}, c.opt.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if r, ok := obj.(*api.Restic); ok && r.Name == c.opt.ResticName && r.IsValid() == nil {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					c.rQueue.Add(key)
				}
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
			if !util.ResticEqual(oldObj, newObj) && newObj.Name == c.opt.ResticName && newObj.IsValid() == nil {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err == nil {
					c.rQueue.Add(key)
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			if r, ok := obj.(*api.Restic); ok && r.Name == c.opt.ResticName {
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					c.rQueue.Add(key)
				}
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	c.rLister = stash_listers.NewResticLister(c.rIndexer)
}

func (c *Scheduler) runResticWatcher() {
	for c.processNextRestic() {
	}
}

func (c *Scheduler) processNextRestic() bool {
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
	err := c.runResticScheduler(key.(string))
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.rQueue.Forget(key)
		return true
	}
	log.Errorf("Failed to process Restic %v. Reason: %s", key, err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.rQueue.NumRequeues(key) < c.opt.MaxNumRequeues {
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
func (c *Scheduler) runResticScheduler(key string) error {
	obj, exists, err := c.rIndexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Restic, so that we will see a delete for one d
		fmt.Printf("Restic %s does not exist anymore\n", key)

		c.cron.Stop()
	} else {
		r := obj.(*api.Restic)
		fmt.Printf("Sync/Add/Update for Restic %s\n", r.GetName())

		c.rchan <- r
		err := c.configureScheduler()
		if err != nil {
			c.recorder.Eventf(
				r.ObjectReference(),
				apiv1.EventTypeWarning,
				eventer.EventReasonFailedToBackup,
				"Failed to start Stash scheduler reason %v",
				err,
			)
			log.Errorln(err)
		}
	}
	return nil
}
