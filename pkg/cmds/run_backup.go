package cmds

import (
	"time"

	"github.com/appscode/go/log"
	cs "github.com/appscode/stash/client/clientset/versioned"
	stashinformers "github.com/appscode/stash/client/informers/externalversions"
	"github.com/appscode/stash/pkg/backup"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/appscode/stash/pkg/restic"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/meta"
)

func NewCmdRunBackup() *cobra.Command {
	con := backup.BackupSessionController{
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
		Short:             "Take backup of workload directories",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := clientcmd.BuildConfigFromFlags(con.MasterURL, con.KubeconfigPath)
			if err != nil {
				glog.Fatalf("Could not get Kubernetes config: %s", err)
				return err
			}

			con.K8sClient = kubernetes.NewForConfigOrDie(config)
			con.StashClient = cs.NewForConfigOrDie(config)
			con.StashInformerFactory = stashinformers.NewSharedInformerFactoryWithOptions(
				con.StashClient,
				con.ResyncPeriod,
				stashinformers.WithNamespace(con.Namespace),
				stashinformers.WithTweakListOptions(nil),
			)
			con.Recorder = eventer.NewEventRecorder(con.K8sClient, backup.BackupEventComponent)
			con.Metrics.JobName = con.BackupConfigurationName
			if err = con.RunBackup(); err != nil {
				log.Errorln("failed to complete backup. Reason: ", err)
				//set BackupSession status "Failed", write event and prometheus metrics
				return con.HandleBackupFailure(err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&con.MasterURL, "master", con.MasterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&con.KubeconfigPath, "kubeconfig", con.KubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&con.BackupConfigurationName, "backup-configuration", con.BackupConfigurationName, "Set BackupConfiguration Name")
	cmd.Flags().StringVar(&con.SetupOpt.SecretDir, "secret-dir", con.SetupOpt.SecretDir, "Directory where storage secret has been mounted")
	cmd.Flags().BoolVar(&con.SetupOpt.EnableCache, "enable-cache", con.SetupOpt.EnableCache, "Specify weather to enable caching for restic")
	cmd.Flags().IntVar(&con.SetupOpt.MaxConnections, "max-connections", con.SetupOpt.MaxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")
	cmd.Flags().BoolVar(&con.Metrics.Enabled, "metrics-enabled", con.Metrics.Enabled, "Specify weather to export Prometheus metrics")
	cmd.Flags().BoolVar(&con.EnableRBAC, "enable-rbac", con.EnableRBAC, "Enable RBAC")
	cmd.Flags().StringVar(&con.Metrics.PushgatewayURL, "pushgateway-url", con.Metrics.PushgatewayURL, "URL of Prometheus pushgateway used to cache backup metrics")

	return cmd
}
