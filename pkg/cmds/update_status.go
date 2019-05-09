package cmds

import (
	"fmt"

	"github.com/appscode/go/flags"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/status"
)

func NewCmdUpdateStatus() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            = status.UpdateStatusOptions{
			OutputFileName: restic.DefaultOutputFileName,
		}
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
			opt.KubeClient, err = kubernetes.NewForConfig(config)
			if err != nil {
				return err
			}
			opt.StashClient, err = cs.NewForConfig(config)
			if err != nil {
				return err
			}

			if opt.BackupSession != "" {
				return opt.UpdateBackupStatusFromFile()
			}
			if opt.RestoreSession != "" {
				return opt.UpdateRestoreStatusFromFile()
			}
			return fmt.Errorf("backup-session or, restore-session not specified")
		},
	}

	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	cmd.Flags().StringVar(&opt.Namespace, "namespace", "default", "Namespace of Backup/Restore Session")
	cmd.Flags().StringVar(&opt.Repository, "repository", opt.Repository, "Name of the Repository")
	cmd.Flags().StringVar(&opt.BackupSession, "backup-session", opt.BackupSession, "Name of the Backup Session")
	cmd.Flags().StringVar(&opt.RestoreSession, "restore-session", opt.RestoreSession, "Name of the Restore Session")
	cmd.Flags().StringVar(&opt.OutputDir, "output-dir", opt.OutputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")

	return cmd
}
