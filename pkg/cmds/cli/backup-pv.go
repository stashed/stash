package cli

import (
	"fmt"
	"strings"

	"github.com/appscode/go/flags"
	"github.com/appscode/go/log"
	"github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewBackupPVCmd() *cobra.Command {
	var (
		kubeConfig string
		namespace  string
		template   string
		volume     string

		targetDirs []string
		mountPath  string
	)

	var cmd = &cobra.Command{
		Use:               "backup-pv",
		Short:             `Backup persistent volume`,
		Long:              `Backup persistent volume using BackupConfiguration Template`,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "volume", "template", "directories", "mountpath")

			c, err := newStashCLIController(kubeConfig)
			if err != nil {
				return err
			}

			// check backupConfigurationTemplate exists
			_, err = c.stashClient.StashV1beta1().BackupConfigurationTemplates().Get(template, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("can't get BackupConfigurationTemplate %s, reason: %s", template, err)
			}

			// get PV
			pv, err := c.kubeClient.CoreV1().PersistentVolumes().Get(volume, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("can't get PersistentVolumes %s, reason: %s", volume, err)
			}

			// create PVC and add default backup annotations
			pvc := &core.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      volume + "-pvc", // use generateName ?
					Namespace: namespace,
					Annotations: map[string]string{
						v1beta1.KeyBackupConfigurationTemplate: template,
						v1beta1.KeyMountPath:                   mountPath,
						v1beta1.KeyTargetDirectories:           strings.Join(targetDirs, ","),
					},
				},
				Spec: core.PersistentVolumeClaimSpec{
					// set other optional fields ?
					VolumeName:  volume,
					AccessModes: pv.Spec.AccessModes,
					Resources: core.ResourceRequirements{
						Limits:   pv.Spec.Capacity,
						Requests: pv.Spec.Capacity,
					},
				},
			}
			pvc, err = c.kubeClient.CoreV1().PersistentVolumeClaims(namespace).Create(pvc)
			if err != nil {
				return err
			}
			log.Infof("PVC %s/%s created and annotated", namespace, pvc.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&kubeConfig, "kubeconfig", kubeConfig, "Path of the Kube config file.")
	cmd.Flags().StringVar(&volume, "volume", volume, "Name of the Persistent volume.")
	cmd.Flags().StringVar(&namespace, "namespace", "default", "Namespace for Persistent Volume Claim.")
	cmd.Flags().StringVar(&template, "template", template, "Name of the BackupConfigurationTemplate.")

	cmd.Flags().StringSliceVar(&targetDirs, "directories", targetDirs, "List of target directories.")
	cmd.Flags().StringVar(&mountPath, "mountpath", mountPath, "Mount path for PVC.")

	return cmd
}
