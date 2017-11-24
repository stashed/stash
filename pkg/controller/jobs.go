package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	batch_listers "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func (c *StashController) initJobWatcher() {
	lw := &cache.ListWatch{ // TODO @ Dipta: only watch stash jobs
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			return c.k8sClient.BatchV1().Jobs(core.NamespaceAll).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.k8sClient.BatchV1().Jobs(core.NamespaceAll).Watch(options)
		},
	}

	// create the workqueue
	c.jobQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "stash-job")

	c.jobIndexer, c.jobInformer = cache.NewIndexerInformer(lw, &batch.Job{}, c.options.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				c.jobQueue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				c.jobQueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				c.jobQueue.Add(key)
			}
		},
	}, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	c.jobLister = batch_listers.NewJobLister(c.jobIndexer)
}

func (c *StashController) runJobWatcher() {
	for c.processNextJob() {
	}
}

func (c *StashController) processNextJob() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.jobQueue.Get()
	if quit {
		return false
	}
	defer c.jobQueue.Done(key)

	// Invoke the method containing the business logic
	err := c.runJobInjector(key.(string))
	if err == nil {
		c.jobQueue.Forget(key)
		return true
	}
	log.Errorf("Failed to process Job %v. Reason: %s", key, err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.jobQueue.NumRequeues(key) < c.options.MaxNumRequeues {
		glog.Infof("Error syncing job %v: %v", key, err)
		c.jobQueue.AddRateLimited(key)
		return true
	}

	c.jobQueue.Forget(key)
	runtime.HandleError(err)
	glog.Infof("Dropping job %q out of the queue: %v", key, err)
	return true
}

func (c *StashController) runJobInjector(key string) error {
	obj, exists, err := c.jobIndexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		fmt.Printf("Job %s does not exist anymore\n", key)
		return nil
	} else {
		job := obj.(*batch.Job)
		fmt.Printf("Sync/Add/Update for Job %s\n", job.GetName())

		if job.Labels["app"] == util.AppLabelStash && job.Status.Succeeded > 0 {
			fmt.Printf("Deleting succeeded job %s\n", job.GetName())
			if err = util.DeleteStashJob(c.k8sClient, *job); err != nil {
				fmt.Printf("Failed to delete stash job: %s, reason: %s\n", job.GetName(), err)
				return err
			}
			fmt.Printf("Deleted stash job: %s\n", job.GetName())
		}
	}
	return nil
}
