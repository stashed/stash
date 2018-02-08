package backup

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func (c *Controller) initResticWatcher() {
	// TODO: Watch one Restic object, when support for Kubernetes 1.8 is dropped.
	// ref: https://github.com/kubernetes/kubernetes/pull/53345

	c.rInformer = c.stashInformerFactory.Stash().V1alpha1().Restics().Informer()
	c.rQueue = queue.New("Restic", c.opt.MaxNumRequeues, c.opt.NumThreads, c.runResticScheduler)
	c.rInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if r, ok := obj.(*api.Restic); ok && r.Name == c.opt.ResticName && r.IsValid() == nil {
				queue.Enqueue(c.rQueue.GetQueue(), obj)
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
				queue.Enqueue(c.rQueue.GetQueue(), newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			queue.Enqueue(c.rQueue.GetQueue(), obj)
		},
	})
	c.rLister = c.stashInformerFactory.Stash().V1alpha1().Restics().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *Controller) runResticScheduler(key string) error {
	obj, exists, err := c.rInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a Restic, so that we will see a delete for one d
		glog.Warningf("Restic %s does not exist anymore\n", key)

		c.cron.Stop()
	} else {
		r := obj.(*api.Restic)
		glog.Infof("Sync/Add/Update for Restic %s\n", r.GetName())

		err := c.configureScheduler(r)
		if err != nil {
			c.recorder.Eventf(
				r.ObjectReference(),
				core.EventTypeWarning,
				eventer.EventReasonFailedToBackup,
				"Failed to start Stash scheduler reason %v",
				err,
			)
			log.Errorln(err)
		}
	}
	return nil
}
