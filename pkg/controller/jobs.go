package controller

import (
	"fmt"
	"time"

	"github.com/appscode/kutil/tools/queue"
	"github.com/appscode/stash/pkg/util"
	"github.com/golang/glog"
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	batch_informers "k8s.io/client-go/informers/batch/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func (c *StashController) initJobWatcher() {
	c.jobInformer = c.kubeInformerFactory.InformerFor(&batch.Job{}, func(client kubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
		return batch_informers.NewFilteredJobInformer(
			client,
			core.NamespaceAll,
			resyncPeriod,
			cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
			func(options *metav1.ListOptions) {
				options.LabelSelector = labels.SelectorFromSet(map[string]string{
					"app": util.AppLabelStash,
				}).String()
			},
		)
	})
	c.jobQueue = queue.New("Job", c.MaxNumRequeues, c.NumThreads, c.runJobInjector)
	c.jobInformer.AddEventHandler(queue.DefaultEventHandler(c.jobQueue.GetQueue()))
	c.jobLister = c.kubeInformerFactory.Batch().V1().Jobs().Lister()
}

func (c *StashController) runJobInjector(key string) error {
	obj, exists, err := c.jobInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("Job %s does not exist anymore\n", key)
		return nil
	} else {
		job := obj.(*batch.Job)
		glog.Infof("Sync/Add/Update for Job %s", job.GetName())

		if job.Status.Succeeded > 0 {
			glog.Infof("Deleting succeeded job %s", job.GetName())

			deletePolicy := metav1.DeletePropagationBackground
			err := c.kubeClient.BatchV1().Jobs(job.Namespace).Delete(job.Name, &metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			})

			if err != nil && !kerr.IsNotFound(err) {
				return fmt.Errorf("failed to delete job: %s, reason: %s", job.Name, err)
			}

			glog.Infof("Deleted stash job: %s", job.GetName())

			if c.EnableRBAC {
				err = c.ensureRepoReaderRolebindingDeleted(&job.ObjectMeta)
				if err != nil {
					return fmt.Errorf("failed to delete repo-reader rolebinding. reason: %s", err)
				}
			}
		}
	}
	return nil
}
