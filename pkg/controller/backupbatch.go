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
	"fmt"

	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	stash_scheme "stash.appscode.dev/stash/client/clientset/versioned/scheme"
	v1beta1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/appscode/go/log"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/reference"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/tools/queue"
)

func (c *StashController) initBackupBatchWatcher() {
	c.bbInformer = c.stashInformerFactory.Stash().V1beta1().BackupBatches().Informer()
	c.bbQueue = queue.New(api_v1beta1.ResourceKindBackupBatch, c.MaxNumRequeues, c.NumThreads, c.runBackupBatchProcessor)
	c.bbInformer.AddEventHandler(queue.NewReconcilableHandler(c.bbQueue.GetQueue()))
	c.bbLister = c.stashInformerFactory.Stash().V1beta1().BackupBatches().Lister()
}

// syncToStdout is the business logic of the controller. In this controller it simply prints
// information about the deployment to stdout. In case an error happened, it has to simply return the error.
// The retry logic should not be part of the business logic.
func (c *StashController) runBackupBatchProcessor(key string) error {
	obj, exists, err := c.bcInformer.GetIndexer().GetByKey(key)
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
	err = c.applyBackupBatchReconciliationLogic(backupBatch)
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

func (c *StashController) applyBackupBatchReconciliationLogic(backupBatch *api_v1beta1.BackupBatch) error {
	// check if BackupBatch is being deleted. if it is being deleted then delete respective resources.
	if backupBatch.DeletionTimestamp != nil {
		if core_util.HasFinalizer(backupBatch.ObjectMeta, api_v1beta1.StashKey) {
			if err := c.EnsureV1beta1SidecarDeleted(backupBatch); err != nil {
				ref, rerr := reference.GetReference(stash_scheme.Scheme, backupBatch)
				if rerr != nil {
					return errors.NewAggregate([]error{err, rerr})
				}
				return c.handleWorkloadControllerTriggerFailure(ref, err)
			}
			if err := c.EnsureCronJobDeletedForBackupBatch(backupBatch); err != nil {
				return err
			}
			// Remove finalizer
			_, _, err := v1beta1_util.PatchBackupBatch(c.stashClient.StashV1beta1(), backupBatch, func(in *api_v1beta1.BackupBatch) *api_v1beta1.BackupBatch {
				in.ObjectMeta = core_util.RemoveFinalizer(in.ObjectMeta, api_v1beta1.StashKey)
				return in

			})
			if err != nil {
				return err
			}
		}
	} else {
		// add a finalizer so that we can remove respective resources before this BackupBatch is deleted
		_, _, err := v1beta1_util.PatchBackupBatch(c.stashClient.StashV1beta1(), backupBatch, func(in *api_v1beta1.BackupBatch) *api_v1beta1.BackupBatch {
			in.ObjectMeta = core_util.AddFinalizer(in.ObjectMeta, api_v1beta1.StashKey)
			return in
		})
		if err != nil {
			return err
		}
		// skip if BackupBatch paused
		if backupBatch.Spec.Paused {
			log.Infof("Skipping processing BackupBatch %s/%s. Reason: Backup Batch is paused.", backupBatch.Namespace, backupBatch.Name)
			return nil
		}
		if len(backupBatch.Spec.BackupConfigurationTemplates) > 0 {
			for _, backupConfigTemp := range backupBatch.Spec.BackupConfigurationTemplates {
				if backupConfigTemp.Spec.Target != nil && backupBatch.Spec.Driver != api_v1beta1.VolumeSnapshotter &&
					util.BackupModel(backupConfigTemp.Spec.Target.Ref.Kind) == util.ModelSidecar {
					if err := c.EnsureV1beta1Sidecar(backupConfigTemp); err != nil {
						ref, rerr := reference.GetReference(stash_scheme.Scheme, backupBatch)
						if rerr != nil {
							return errors.NewAggregate([]error{err, rerr})
						}
						return c.handleWorkloadControllerTriggerFailure(ref, err)
					}
				}
			}
			// create a CronJob that will create BackupSession on each schedule
			err = c.EnsureCronJobForBackupBatch(backupBatch)
		}
		if err != nil {
			return c.handleCronJobCreationFailure(backupBatch, err)
		}
	}
	return nil
}

// EnsureCronJob creates a Kubernetes CronJob for a BackupBatch object
func (c *StashController) EnsureCronJobForBackupBatch(backupBatch *api_v1beta1.BackupBatch) error {
	if backupBatch == nil {
		return fmt.Errorf("BackupBatch is nil")
	}
	ref, err := reference.GetReference(stash_scheme.Scheme, backupBatch)
	if err != nil {
		return err
	}

	return c.EnsureCronJob(ref, backupBatch.Spec.RuntimeSettings.Pod, backupBatch.OffshootLabels(), backupBatch.Spec.Schedule)
}

func (c *StashController) EnsureCronJobDeletedForBackupBatch(backupBatch *api_v1beta1.BackupBatch) error {
	ref, err := reference.GetReference(stash_scheme.Scheme, backupBatch)
	if err != nil {
		return err
	}
	return c.EnsureCronJobDeleted(backupBatch.ObjectMeta, ref)
}
