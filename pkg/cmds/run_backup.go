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
package cmds

import (
	"time"

	cs "stash.appscode.dev/stash/client/clientset/versioned"
	stashinformers "stash.appscode.dev/stash/client/informers/externalversions"
	"stash.appscode.dev/stash/pkg/backup"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/util"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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
				glog.Fatalf("Could not get Kubernetes config: %s", err)
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
			opt.Metrics.JobName = opt.BackupConfigurationName
			opt.Host, err = util.GetBackupHostName(opt.StashClient, opt.BackupConfigurationName, opt.Namespace)
			if err != nil {
				return err
			}
			// run backup
			if err = opt.RunBackup(); err != nil {
				return opt.HandleBackupSetupFailure(err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opt.MasterURL, "master", opt.MasterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&opt.KubeconfigPath, "kubeconfig", opt.KubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.BackupConfigurationName, "backupconfiguration", opt.BackupConfigurationName, "Name of the respective BackupConfiguration object")
	cmd.Flags().StringVar(&opt.SetupOpt.SecretDir, "secret-dir", opt.SetupOpt.SecretDir, "Directory where storage secret has been mounted")
	cmd.Flags().BoolVar(&opt.SetupOpt.EnableCache, "enable-cache", opt.SetupOpt.EnableCache, "Specify whether to enable caching for restic")
	cmd.Flags().IntVar(&opt.SetupOpt.MaxConnections, "max-connections", opt.SetupOpt.MaxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")
	cmd.Flags().BoolVar(&opt.Metrics.Enabled, "metrics-enabled", opt.Metrics.Enabled, "Specify whether to export Prometheus metrics")
	cmd.Flags().StringVar(&opt.Metrics.PushgatewayURL, "pushgateway-url", opt.Metrics.PushgatewayURL, "URL of Prometheus pushgateway used to cache backup metrics")

	return cmd
}
