/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	stash_rbac "stash.appscode.dev/stash/pkg/rbac"

	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	batch_informers "k8s.io/client-go/informers/batch/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/tools/queue"
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
					apis.KeyDeleteJobOnCompletion: apis.AllowDeletingJobOnCompletion,
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
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		klog.Warningf("Job %s does not exist anymore\n", key)
		return nil
	} else {
		job := obj.(*batch.Job)
		klog.Infof("Sync/Add/Update for Job %s", job.GetName())

		if job.Status.Succeeded > 0 {
			klog.Infof("Deleting succeeded job %s", job.GetName())

			deletePolicy := metav1.DeletePropagationBackground
			err := c.kubeClient.BatchV1().Jobs(job.Namespace).Delete(context.TODO(), job.Name, metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			})

			if err != nil && !kerr.IsNotFound(err) {
				return fmt.Errorf("failed to delete job: %s, reason: %s", job.Name, err)
			}

			klog.Infof("Deleted stash job: %s", job.GetName())

			err = stash_rbac.EnsureRepoReaderRolebindingDeleted(c.kubeClient, c.stashClient, &job.ObjectMeta)
			if err != nil {
				return fmt.Errorf("failed to delete repo-reader rolebinding. reason: %s", err)
			}

		}
	}
	return nil
}
