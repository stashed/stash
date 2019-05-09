package cli

import (
	"fmt"

	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"
	core_util "kmodules.xyz/client-go/core/v1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	stash_scheme "stash.appscode.dev/stash/client/clientset/versioned/scheme"
	"stash.appscode.dev/stash/pkg/util"
)

func NewTriggerBackupCmd() *cobra.Command {
	var (
		kubeConfig       string
		namespace        string
		backupConfigName string // from flags or args ?
	)

	var cmd = &cobra.Command{
		Use:               "trigger-backup",
		Short:             `Trigger a backup`,
		Long:              `Trigger a backup by creating BackupSession`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "" {
				return fmt.Errorf("BackupConfiguration name not found")
			}
			backupConfigName = args[0]

			c, err := newStashCLIController(kubeConfig)
			if err != nil {
				return err
			}

			// get backupConfiguration
			backupConfig, err := c.stashClient.StashV1beta1().BackupConfigurations(namespace).Get(backupConfigName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			// create backupSession for backupConfig
			backupSession := &v1beta1.BackupSession{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: backupConfigName + "-",
					Namespace:    namespace,
					Labels: map[string]string{
						util.LabelApp:                 util.AppLabelStash,
						util.LabelBackupConfiguration: backupConfigName,
					},
				},
				Spec: v1beta1.BackupSessionSpec{
					BackupConfiguration: v1.LocalObjectReference{
						Name: backupConfigName,
					},
				},
			}

			// set backupConfig as backupSession's owner
			ref, err := reference.GetReference(stash_scheme.Scheme, backupConfig)
			if err != nil {
				return err
			}
			core_util.EnsureOwnerReference(&backupSession.ObjectMeta, ref)

			// don't use createOrPatch here
			backupSession, err = c.stashClient.StashV1beta1().BackupSessions(namespace).Create(backupSession)
			if err != nil {
				return err
			}

			log.Infof("BackupSession %s/%s created", backupSession.Namespace, backupSession.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&kubeConfig, "kubeconfig", kubeConfig, "Path of the Kube config file.")
	cmd.Flags().StringVar(&namespace, "namespace", "default", "Namespace of the Repository.")
	// cmd.Flags().StringVar(&backupConfigName, "backup-configuration", backupConfigName, "Name of the BackupConfiguration.")

	return cmd
}
