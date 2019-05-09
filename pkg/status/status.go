package status

import (
	"fmt"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/apis/core"
	"stash.appscode.dev/stash/apis"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	stash_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
	stash_util_v1beta1 "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/restic"
)

type UpdateStatusOptions struct {
	KubeClient  kubernetes.Interface
	StashClient *cs.Clientset

	Namespace      string
	Repository     string
	BackupSession  string
	RestoreSession string
	OutputDir      string
	OutputFileName string
}

func (o UpdateStatusOptions) UpdateBackupStatusFromFile() error {
	// read backup output from file
	backupOutput, err := restic.ReadBackupOutput(filepath.Join(o.OutputDir, o.OutputFileName))
	if err != nil {
		return err
	}
	var backupErr error
	if backupOutput.HostBackupStats.Error != "" {
		backupErr = fmt.Errorf(backupOutput.HostBackupStats.Error)
	}
	updateStatusErr := o.UpdatePostBackupStatus(backupOutput)
	return errors.NewAggregate([]error{backupErr, updateStatusErr})
}

func (o UpdateStatusOptions) UpdateRestoreStatusFromFile() error {
	// read restore output from file
	restoreOutput, err := restic.ReadRestoreOutput(filepath.Join(o.OutputDir, o.OutputFileName))
	if err != nil {
		return err
	}
	var restoreErr error
	if restoreOutput.HostRestoreStats.Error != "" {
		restoreErr = fmt.Errorf(restoreOutput.HostRestoreStats.Error)
	}
	updateStatusErr := o.UpdatePostRestoreStatus(restoreOutput)
	return errors.NewAggregate([]error{restoreErr, updateStatusErr})
}

func (o UpdateStatusOptions) UpdatePostBackupStatus(backupOutput *restic.BackupOutput) error {
	// get backup session, update status and create event
	backupSession, err := o.StashClient.StashV1beta1().BackupSessions(o.Namespace).Get(o.BackupSession, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// add or update entry for this host in BackupSession status
	_, err = stash_util_v1beta1.UpdateBackupSessionStatusForHost(o.StashClient.StashV1beta1(), backupSession, backupOutput.HostBackupStats)
	if err != nil {
		return err
	}

	// create event for backup session
	var eventType, eventReason, eventMessage string
	if backupOutput.HostBackupStats.Error != "" {
		eventType = core.EventTypeWarning
		eventReason = eventer.EventReasonHostBackupFailed
		eventMessage = fmt.Sprintf("backup failed for host %q. Reason: %s", backupOutput.HostBackupStats.Hostname, backupOutput.HostBackupStats.Error)
	} else {
		eventType = core.EventTypeNormal
		eventReason = eventer.EventReasonHostBackupSucceded
		eventMessage = fmt.Sprintf("backup succeeded for host %s", backupOutput.HostBackupStats.Hostname)
	}
	_, err = eventer.CreateEvent(
		o.KubeClient,
		eventer.BackupSessionEventComponent,
		backupSession,
		eventType,
		eventReason,
		eventMessage,
	)
	if err != nil {
		return err
	}

	// no need to update repository status for failed backup
	if backupOutput.HostBackupStats.Error != "" {
		return nil
	}

	// get repository and update status
	repository, err := o.StashClient.StashV1alpha1().Repositories(o.Namespace).Get(o.Repository, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, err = stash_util.UpdateRepositoryStatus(
		o.StashClient.StashV1alpha1(),
		repository,
		func(in *api.RepositoryStatus) *api.RepositoryStatus {
			// TODO: fix Restic Wrapper
			in.Integrity = backupOutput.RepositoryStats.Integrity
			in.Size = backupOutput.RepositoryStats.Size
			in.SnapshotCount = backupOutput.RepositoryStats.SnapshotCount
			in.SnapshotRemovedOnLastCleanup = backupOutput.RepositoryStats.SnapshotRemovedOnLastCleanup

			currentTime := metav1.Now()
			in.LastBackupTime = &currentTime

			if in.FirstBackupTime == nil {
				in.FirstBackupTime = &currentTime
			}
			return in
		},
		apis.EnableStatusSubresource,
	)
	return err
}

func (o UpdateStatusOptions) UpdatePostRestoreStatus(restoreOutput *restic.RestoreOutput) error {
	// get restore session, update status and create event
	restoreSession, err := o.StashClient.StashV1beta1().RestoreSessions(o.Namespace).Get(o.RestoreSession, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// add or update entry for this host in RestoreSession status
	_, err = stash_util_v1beta1.UpdateRestoreSessionStatusForHost(o.StashClient.StashV1beta1(), restoreSession, restoreOutput.HostRestoreStats)
	if err != nil {
		return err
	}

	// create event for restore session
	var eventType, eventReason, eventMessage string
	if restoreOutput.HostRestoreStats.Error != "" {
		eventType = core.EventTypeWarning
		eventReason = eventer.EventReasonHostRestoreFailed
		eventMessage = fmt.Sprintf("restore failed for host %q. Reason: %s", restoreOutput.HostRestoreStats.Hostname, restoreOutput.HostRestoreStats.Error)
	} else {
		eventType = core.EventTypeNormal
		eventReason = eventer.EventReasonHostRestoreSucceeded
		eventMessage = fmt.Sprintf("restore succeeded for host %q", restoreOutput.HostRestoreStats.Hostname)
	}
	_, err = eventer.CreateEvent(
		o.KubeClient,
		eventer.RestoreSessionEventComponent,
		restoreSession,
		eventType,
		eventReason,
		eventMessage,
	)
	return err
}
