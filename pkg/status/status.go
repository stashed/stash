package status

import (
	"path/filepath"

	"github.com/appscode/stash/apis"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	cs "github.com/appscode/stash/client/clientset/versioned"
	stash_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
	stash_util_v1beta1 "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"github.com/appscode/stash/pkg/restic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type UpdateStatusOptions struct {
	Namespace      string
	Repository     string
	BackupSession  string
	RestoreSession string
	OutputDir      string
	OutputFileName string
}

func (o UpdateStatusOptions) UpdatePostBackupStatus(client *cs.Clientset) error {
	// read backup output from file
	backupOutput, err := restic.ReadBackupOutput(filepath.Join(o.OutputDir, o.OutputFileName))
	if err != nil {
		return err
	}

	// get backup session and update status
	backupSession, err := client.StashV1beta1().BackupSessions(o.Namespace).Get(o.BackupSession, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, err = stash_util_v1beta1.UpdateBackupSessionStatus(
		client.StashV1beta1(),
		backupSession,
		func(in *api_v1beta1.BackupSessionStatus) *api_v1beta1.BackupSessionStatus {
			in.Phase = api_v1beta1.BackupSessionSucceeded
			in.Stats = backupOutput.BackupStats
			return in
		},
		apis.EnableStatusSubresource,
	)
	if err != nil {
		return err
	}

	// get repository and update status
	repository, err := client.StashV1alpha1().Repositories(o.Namespace).Get(o.Repository, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, err = stash_util.UpdateRepositoryStatus(
		client.StashV1alpha1(),
		repository,
		func(in *api.RepositoryStatus) *api.RepositoryStatus {
			// TODO: fix API
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

func (o UpdateStatusOptions) UpdatePostRestoreStatus(client *cs.Clientset) error {
	// read restore output from file
	restoreOutput, err := restic.ReadRestoreOutput(filepath.Join(o.OutputDir, o.OutputFileName))
	if err != nil {
		return err
	}

	// get restore session and update status
	restoreSession, err := client.StashV1beta1().RestoreSessions(o.Namespace).Get(o.RestoreSession, metav1.GetOptions{})
	if err != nil {
		return err
	}
	_, err = stash_util_v1beta1.UpdateRestoreSessionStatus(
		client.StashV1beta1(),
		restoreSession,
		func(in *api_v1beta1.RestoreSessionStatus) *api_v1beta1.RestoreSessionStatus {
			in.Phase = api_v1beta1.RestoreSucceeded
			in.Duration = restoreOutput.SessionDuration
			return in
		},
		apis.EnableStatusSubresource,
	)

	return err
}
