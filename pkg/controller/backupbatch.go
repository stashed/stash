/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"stash.appscode.dev/apimachinery/apis"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"

	"github.com/golang/glog"
	"kmodules.xyz/client-go/tools/queue"
)

func (c *StashController) initBackupBatchWatcher() {
	c.backupBatchInformer = c.stashInformerFactory.Stash().V1beta1().BackupBatches().Informer()
	c.backupBatchQueue = queue.New(api_v1beta1.ResourceKindBackupBatch, c.MaxNumRequeues, c.NumThreads, c.runBackupBatchProcessor)
	c.backupBatchInformer.AddEventHandler(queue.NewReconcilableHandler(c.backupBatchQueue.GetQueue()))
	c.backupBatchLister = c.stashInformerFactory.Stash().V1beta1().BackupBatches().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runBackupBatchProcessor(key string) error {
	obj, exists, err := c.backupBatchInformer.GetIndexer().GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	if !exists {
		glog.Warningf("BackupBatch %s does not exit anymore\n", key)
		return nil
	}

	backupBatch := obj.(*api_v1beta1.BackupBatch)
	glog.Infof("Sync/Add/Update for BackupBatch %s", backupBatch.GetName())
	// process syc/add/update event
	invoker, err := apis.ExtractBackupInvokerInfo(c.stashClient, api_v1beta1.ResourceKindBackupBatch, backupBatch.Name, backupBatch.Namespace)
	if err != nil {
		return err
	}
	err = c.applyBackupInvokerReconciliationLogic(invoker)
	if err != nil {
		return err
	}

	// We have successfully completed respective stuffs for the current state of this resource.
	// Hence, let's set observed generation as same as the current generation.
	_, err = v1beta1_util.UpdateBackupBatchStatus(c.stashClient.StashV1beta1(), backupBatch, func(in *api_v1beta1.BackupBatchStatus) *api_v1beta1.BackupBatchStatus {
		in.ObservedGeneration = backupBatch.Generation
		return in
	})

	return err
}
