package status

import (
	"fmt"
	"path/filepath"

	"github.com/appscode/stash/apis"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	cs "github.com/appscode/stash/client/clientset/versioned"
	stash_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
	stash_util_v1beta1 "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/restic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/apis/core"
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
	if backupOutput.Error != "" {
		backupErr = fmt.Errorf(backupOutput.Error)
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
	if restoreOutput.Error != "" {
		restoreErr = fmt.Errorf(restoreOutput.Error)
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
	_, err = stash_util_v1beta1.UpdateBackupSessionStatus(
		o.StashClient.StashV1beta1(),
		backupSession,
		func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
			if backupOutput.Error != "" {
				in.Phase = api_v1beta1.BackupSessionFailed
			} else {
				in.Phase = api_v1beta1.BackupSessionSucceeded
				in.Stats = backupOutput.BackupStats
			}
			return in
		},
		apis.EnableStatusSubresource,
	)
	if err != nil {
		return err
	}

	// create event for backup session
	var eventType, eventReason, eventMessage string
	if backupOutput.Error != "" {
		eventType = core.EventTypeWarning
		eventReason = eventer.EventReasonBackupSessionFailed
		eventMessage = fmt.Sprintf("backup session failed, reason: %s", backupOutput.Error)
	} else {
		eventType = core.EventTypeNormal
		eventReason = eventer.EventReasonBackupSessionSucceeded
		eventMessage = "backup session succeeded"
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
	if backupOutput.Error != "" {
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
	_, err = stash_util_v1beta1.UpdateRestoreSessionStatus(
		o.StashClient.StashV1beta1(),
		restoreSession,
		func(in *api_v1beta1.RestoreSessionStatus) *api_v1beta1.RestoreSessionStatus {
			if restoreOutput.Error != "" {
				in.Phase = api_v1beta1.RestoreFailed
			} else {
				in.Phase = api_v1beta1.RestoreSucceeded
			}
			in.Duration = restoreOutput.SessionDuration
			return in
		},
		apis.EnableStatusSubresource,
	)
	if err != nil {
		return err
	}

	// create event for restore session
	var eventType, eventReason, eventMessage string
	if restoreOutput.Error != "" {
		eventType = core.EventTypeWarning
		eventReason = eventer.EventReasonRestoreSessionFailed
		eventMessage = fmt.Sprintf("restore session failed, reason: %s", restoreOutput.Error)
	} else {
		eventType = core.EventTypeNormal
		eventReason = eventer.EventReasonRestoreSessionSucceeded
		eventMessage = "restore session succeeded"
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
