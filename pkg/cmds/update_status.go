/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmds

import (
	"context"
	"fmt"
	"strings"

	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/status"

	"github.com/spf13/cobra"
	"gomodules.xyz/flags"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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

			opt.Config = config
			opt.Metrics.JobName = fmt.Sprintf("%s-%s-%s", strings.ToLower(opt.InvokerKind), opt.Namespace, opt.InvokerName)

			repo, err := opt.StashClient.StashV1alpha1().Repositories(opt.Repository.Namespace).Get(context.Background(), opt.Repository.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			opt.SetupOpt.StorageSecret, err = opt.KubeClient.CoreV1().Secrets(repo.Namespace).Get(context.Background(), repo.Spec.Backend.StorageSecretName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if opt.BackupSession != "" {
				return opt.UpdateBackupStatusFromFile()
			} else {
				return opt.UpdateRestoreStatusFromFile()
			}
		},
	}

	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	cmd.Flags().StringVar(&opt.SetupOpt.Provider, "provider", opt.SetupOpt.Provider, "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&opt.SetupOpt.Bucket, "bucket", opt.SetupOpt.Bucket, "Name of the cloud bucket/container (keep empty for local backend)")
	cmd.Flags().StringVar(&opt.SetupOpt.Endpoint, "endpoint", opt.SetupOpt.Endpoint, "Endpoint for s3/s3 compatible backend or REST server URL")
	cmd.Flags().StringVar(&opt.SetupOpt.Region, "region", opt.SetupOpt.Region, "Region for s3/s3 compatible backend")
	cmd.Flags().StringVar(&opt.SetupOpt.Path, "path", opt.SetupOpt.Path, "Directory inside the bucket where backed up data will be stored")
	cmd.Flags().StringVar(&opt.SetupOpt.ScratchDir, "scratch-dir", opt.SetupOpt.ScratchDir, "Temporary directory")
	cmd.Flags().BoolVar(&opt.SetupOpt.EnableCache, "enable-cache", opt.SetupOpt.EnableCache, "Specify whether to enable caching for restic")
	cmd.Flags().Int64Var(&opt.SetupOpt.MaxConnections, "max-connections", opt.SetupOpt.MaxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")
	cmd.Flags().StringVar(&opt.Namespace, "namespace", "default", "Namespace of Backup/Restore Session")
	cmd.Flags().StringVar(&opt.Repository.Name, "repo-name", opt.Repository.Name, "Name of the Repository")
	cmd.Flags().StringVar(&opt.Repository.Namespace, "repo-namespace", opt.Repository.Namespace, "Namespace of the Repository")
	cmd.Flags().StringVar(&opt.InvokerKind, "invoker-kind", opt.InvokerKind, "Type of the respective backup/restore invoker")
	cmd.Flags().StringVar(&opt.InvokerName, "invoker-name", opt.InvokerName, "Name of the respective backup/restore invoker")
	cmd.Flags().StringVar(&opt.TargetRef.Kind, "target-kind", "", "Kind of the target")
	cmd.Flags().StringVar(&opt.TargetRef.Name, "target-name", "", "Name of the target")
	cmd.Flags().StringVar(&opt.BackupSession, "backupsession", opt.BackupSession, "Name of the Backup Session")
	cmd.Flags().StringVar(&opt.OutputDir, "output-dir", opt.OutputDir, "Directory where output.json file will be written (keep empty if you don't need to write output in file)")
	cmd.Flags().BoolVar(&opt.Metrics.Enabled, "metrics-enabled", opt.Metrics.Enabled, "Specify whether to export Prometheus metrics")
	cmd.Flags().StringVar(&opt.Metrics.PushgatewayURL, "metrics-pushgateway-url", opt.Metrics.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	cmd.Flags().StringSliceVar(&opt.Metrics.Labels, "metrics-labels", opt.Metrics.Labels, "Labels to apply in exported metrics")

	return cmd
}
