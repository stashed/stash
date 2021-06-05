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
	"fmt"
	"strings"
	"time"

	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stashinformers "stash.appscode.dev/apimachinery/client/informers/externalversions"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/apimachinery/pkg/restic"
	"stash.appscode.dev/stash/pkg/backup"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"kmodules.xyz/client-go/meta"
)

func NewCmdRunBackup() *cobra.Command {
	opt := backup.BackupSessionController{
		MasterURL:      "",
		KubeconfigPath: "",
		Namespace:      meta.Namespace(),
		MaxNumRequeues: 5,
		NumThreads:     1,
		ResyncPeriod:   5 * time.Minute,
		SetupOpt: restic.SetupOptions{
			ScratchDir:  restic.DefaultScratchDir,
			EnableCache: true,
		},
	}

	cmd := &cobra.Command{
		Use:               "run-backup",
		Short:             "Take backup of workload paths",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := clientcmd.BuildConfigFromFlags(opt.MasterURL, opt.KubeconfigPath)
			if err != nil {
				klog.Fatalf("Could not get Kubernetes config: %s", err)
				return err
			}

			opt.Config = config
			opt.K8sClient = kubernetes.NewForConfigOrDie(config)
			opt.StashClient = cs.NewForConfigOrDie(config)
			opt.StashInformerFactory = stashinformers.NewSharedInformerFactoryWithOptions(
				opt.StashClient,
				opt.ResyncPeriod,
				stashinformers.WithNamespace(opt.Namespace),
				stashinformers.WithTweakListOptions(nil),
			)
			opt.Recorder = eventer.NewEventRecorder(opt.K8sClient, backup.BackupEventComponent)
			opt.Metrics.JobName = fmt.Sprintf("%s-%s-%s", strings.ToLower(opt.InvokerKind), opt.Namespace, opt.InvokerName)

			inv, err := invoker.ExtractBackupInvokerInfo(opt.StashClient, opt.InvokerKind, opt.InvokerName, opt.Namespace)
			if err != nil {
				return err
			}

			for _, targetInfo := range inv.TargetsInfo {
				if targetInfo.Target != nil && targetMatched(targetInfo.Target.Ref, opt.BackupTargetKind, opt.BackupTargetName) {

					opt.Host, err = util.GetHostName(targetInfo.Target)
					if err != nil {
						return err
					}

					// run backup
					err = opt.RunBackup(targetInfo, inv.ObjectRef)
					if err != nil {
						return opt.HandleBackupSetupFailure(inv.ObjectRef, err)
					}
				}
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&opt.MasterURL, "master", opt.MasterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&opt.KubeconfigPath, "kubeconfig", opt.KubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.InvokerKind, "invoker-kind", opt.InvokerKind, "Kind of the backup invoker")
	cmd.Flags().StringVar(&opt.InvokerName, "invoker-name", opt.InvokerName, "Name of the respective backup invoker")
	cmd.Flags().StringVar(&opt.BackupTargetKind, "target-kind", opt.BackupTargetKind, "Kind of the Target")
	cmd.Flags().StringVar(&opt.BackupTargetName, "target-name", opt.BackupTargetName, "Name of the Target")
	cmd.Flags().StringVar(&opt.Host, "host", opt.Host, "Name of the host that will be backed up")
	cmd.Flags().StringVar(&opt.SetupOpt.SecretDir, "secret-dir", opt.SetupOpt.SecretDir, "Directory where storage secret has been mounted")
	cmd.Flags().BoolVar(&opt.SetupOpt.EnableCache, "enable-cache", opt.SetupOpt.EnableCache, "Specify whether to enable caching for restic")
	cmd.Flags().Int64Var(&opt.SetupOpt.MaxConnections, "max-connections", opt.SetupOpt.MaxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")
	cmd.Flags().BoolVar(&opt.Metrics.Enabled, "metrics-enabled", opt.Metrics.Enabled, "Specify whether to export Prometheus metrics")
	cmd.Flags().StringVar(&opt.Metrics.PushgatewayURL, "pushgateway-url", opt.Metrics.PushgatewayURL, "URL of Prometheus pushgateway used to cache backup metrics")

	return cmd
}

func targetMatched(tref v1beta1.TargetRef, expectedKind, expectedName string) bool {
	return tref.Kind == expectedKind && tref.Name == expectedName
}
