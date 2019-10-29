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
package status

import (
	"fmt"
	"path/filepath"

	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	stash_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
	stash_util_v1beta1 "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/restic"

	"github.com/appscode/go/log"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type UpdateStatusOptions struct {
	Config      *rest.Config
	KubeClient  kubernetes.Interface
	StashClient cs.Interface

	Namespace      string
	Repository     string
	BackupSession  string
	RestoreSession string
	OutputDir      string
	OutputFileName string
	Metrics        restic.MetricsOptions
}

func (o UpdateStatusOptions) UpdateBackupStatusFromFile() error {
	// read backup output from file
	log.Infof("Reading backup output from file: %s", filepath.Join(o.OutputDir, o.OutputFileName))
	backupOutput, err := restic.ReadBackupOutput(filepath.Join(o.OutputDir, o.OutputFileName))
	if err != nil {
		return err
	}
	return o.UpdatePostBackupStatus(backupOutput)
}

func (o UpdateStatusOptions) UpdateRestoreStatusFromFile() error {
	// read restore output from file
	log.Infof("Reading restore output from file: %s", filepath.Join(o.OutputDir, o.OutputFileName))
	restoreOutput, err := restic.ReadRestoreOutput(filepath.Join(o.OutputDir, o.OutputFileName))
	if err != nil {
		return err
	}
	return o.UpdatePostRestoreStatus(restoreOutput)
}

func (o UpdateStatusOptions) UpdatePostBackupStatus(backupOutput *restic.BackupOutput) error {
	if backupOutput == nil {
		return fmt.Errorf("invalid backup ouputput. Backup output must not be nil")
	}
	// get backup session, update status and create event
	backupSession, err := o.StashClient.StashV1beta1().BackupSessions(o.Namespace).Get(o.BackupSession, metav1.GetOptions{})
	if err != nil {
		return err
	}

	overallBackupSucceeded := true

	// add or update entry for each host in BackupSession status + create event
	for _, hostStats := range backupOutput.HostBackupStats {
		log.Infof("Updating status of BackupSession: %s/%s for host: %s", backupSession.Namespace, backupSession.Name, hostStats.Hostname)
		backupSession, err = stash_util_v1beta1.UpdateBackupSessionStatusForHost(o.StashClient.StashV1beta1(), backupSession, hostStats)
		if err != nil {
			return err
		}
		// create event to the BackupSession
		var eventType, eventReason, eventMessage string
		if hostStats.Error != "" {
			overallBackupSucceeded = false
			eventType = core.EventTypeWarning
			eventReason = eventer.EventReasonHostBackupFailed
			eventMessage = fmt.Sprintf("backup failed for host %q. Reason: %s", hostStats.Hostname, hostStats.Error)
		} else {
			eventType = core.EventTypeNormal
			eventReason = eventer.EventReasonHostBackupSucceded
			eventMessage = fmt.Sprintf("backup succeeded for host %s", hostStats.Hostname)
		}
		_, err = eventer.CreateEvent(
			o.KubeClient,
			eventer.EventSourceStatusUpdater,
			backupSession,
			eventType,
			eventReason,
			eventMessage,
		)
		if err != nil {
			return err
		}
	}

	// if overall backup succeeded and repository status presents in backupOutput then update Repository status
	if overallBackupSucceeded && backupOutput.RepositoryStats.Integrity != nil {
		repository, err := o.StashClient.StashV1alpha1().Repositories(o.Namespace).Get(o.Repository, metav1.GetOptions{})
		if err != nil {
			return err
		}

		_, err = stash_util.UpdateRepositoryStatus(
			o.StashClient.StashV1alpha1(),
			repository,
			func(in *api.RepositoryStatus) *api.RepositoryStatus {
				in.Integrity = backupOutput.RepositoryStats.Integrity
				in.Size = backupOutput.RepositoryStats.Size
				in.SnapshotCount = backupOutput.RepositoryStats.SnapshotCount
				in.SnapshotsRemovedOnLastCleanup = backupOutput.RepositoryStats.SnapshotsRemovedOnLastCleanup

				currentTime := metav1.Now()
				in.LastBackupTime = &currentTime

				if in.FirstBackupTime == nil {
					in.FirstBackupTime = &currentTime
				}
				return in
			},
		)
		if err != nil {
			return err
		}
	}
	// if metrics enabled then send metrics to the Prometheus pushgateway
	if o.Metrics.Enabled {
		backupConfig, err := o.StashClient.StashV1beta1().BackupConfigurations(o.Namespace).Get(backupSession.Spec.Invoker.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		return o.Metrics.SendBackupHostMetrics(o.Config, backupConfig, backupOutput)
	}
	return nil
}

func (o UpdateStatusOptions) UpdatePostRestoreStatus(restoreOutput *restic.RestoreOutput) error {
	if restoreOutput == nil {
		return fmt.Errorf("invalid restore output. Restore output must not be nil")
	}
	// get restore session, update status and create event
	restoreSession, err := o.StashClient.StashV1beta1().RestoreSessions(o.Namespace).Get(o.RestoreSession, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// add or update entry for each host in RestoreSession status
	for _, hostStats := range restoreOutput.HostRestoreStats {
		log.Infof("Updating status of RestoreSession: %s/%s for host: %s", restoreSession.Namespace, restoreSession.Name, hostStats.Hostname)
		restoreSession, err = stash_util_v1beta1.UpdateRestoreSessionStatusForHost(o.StashClient.StashV1beta1(), restoreSession, hostStats)
		if err != nil {
			return err
		}

		// create event to the RestoreSession
		var eventType, eventReason, eventMessage string
		if hostStats.Error != "" {
			eventType = core.EventTypeWarning
			eventReason = eventer.EventReasonHostRestoreFailed
			eventMessage = fmt.Sprintf("restore failed for host %q. Reason: %s", hostStats.Hostname, hostStats.Error)
		} else {
			eventType = core.EventTypeNormal
			eventReason = eventer.EventReasonHostRestoreSucceeded
			eventMessage = fmt.Sprintf("restore succeeded for host %q", hostStats.Hostname)
		}
		_, err = eventer.CreateEvent(
			o.KubeClient,
			eventer.EventSourceStatusUpdater,
			restoreSession,
			eventType,
			eventReason,
			eventMessage,
		)
		if err != nil {
			return err
		}
	}
	// if metrics enabled then send metrics to the Prometheus pushgateway
	if o.Metrics.Enabled {
		return o.Metrics.SendRestoreHostMetrics(o.Config, restoreSession, restoreOutput)
	}
	return nil
}
