package cmds

import (
	"fmt"
	"path/filepath"

	"github.com/appscode/go/flags"
	"github.com/appscode/stash/apis"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	cs "github.com/appscode/stash/client/clientset/versioned"
	stash_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1alpha1/util"
	stash_util_v1beta1 "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"github.com/appscode/stash/pkg/restic"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

type updateStatusOptions struct {
	namespace      string
	repository     string
	backupSession  string
	restoreSession string
	outputDir      string
}

func NewCmdUpdateStatus() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            updateStatusOptions
	)

	cmd := &cobra.Command{
		Use:               "update-status",
		Short:             "Update status of Repository, Backup/Restore Session",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "namespace", "output-dir")

			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				return err
			}
			stashClient, err := cs.NewForConfig(config)
			if err != nil {
				return err
			}

			// TODO: fix for failed backup/restore
			if opt.backupSession != "" {
				return opt.updateStatusForBackupSession(stashClient)
			}
			if opt.restoreSession != "" {
				return opt.updateStatusForRestoreSession(stashClient)
			}
			return fmt.Errorf("backup-session or, restore-session not specified")
		},
	}

	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	cmd.Flags().StringVar(&opt.namespace, "namespace", "default", "Namespace of Backup/Restore Session")
	cmd.Flags().StringVar(&opt.repository, "repository", opt.repository, "Name of the Repository")
	cmd.Flags().StringVar(&opt.backupSession, "backup-session", opt.backupSession, "Name of the Backup Session")
	cmd.Flags().StringVar(&opt.restoreSession, "restore-session", opt.restoreSession, "Name of the Restore Session")
	cmd.Flags().StringVar(&opt.outputDir, "output-dir", opt.outputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")

	return cmd
}

func (o updateStatusOptions) updateStatusForBackupSession(client *cs.Clientset) error {
	// read backup output from file
	backupOutput, err := restic.ReadBackupOutput(filepath.Join(o.outputDir, restic.DefaultOutputFileName))
	if err != nil {
		return err
	}

	// get backup session and update status
	backupSession, err := client.StashV1beta1().BackupSessions(o.namespace).Get(o.backupSession, metav1.GetOptions{})
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
	repository, err := client.StashV1alpha1().Repositories(o.namespace).Get(o.repository, metav1.GetOptions{})
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

func (o updateStatusOptions) updateStatusForRestoreSession(client *cs.Clientset) error {
	// read restore output from file
	restoreOutput, err := restic.ReadRestoreOutput(filepath.Join(o.outputDir, restic.DefaultOutputFileName))
	if err != nil {
		return err
	}

	// get restore session and update status
	restoreSession, err := client.StashV1beta1().RestoreSessions(o.namespace).Get(o.restoreSession, metav1.GetOptions{})
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
