package cmds

import (
	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/meta"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/restic"
	"stash.appscode.dev/stash/pkg/restore"
	"stash.appscode.dev/stash/pkg/util"
)

func NewCmdRestore() *cobra.Command {
	opt := &restore.Options{
		MasterURL:      "",
		KubeconfigPath: "",
		Namespace:      meta.Namespace(),
		SetupOpt: restic.SetupOptions{
			ScratchDir:  "/tmp",
			EnableCache: true,
		},
		RestoreModel: restore.RestoreModelInitContainer,
	}

	cmd := &cobra.Command{
		Use:               "restore",
		Short:             "Restore from backup",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// create client
			config, err := clientcmd.BuildConfigFromFlags(opt.MasterURL, opt.KubeconfigPath)
			if err != nil {
				log.Fatal(err)
				return err
			}
			opt.KubeClient = kubernetes.NewForConfigOrDie(config)
			opt.StashClient = cs.NewForConfigOrDie(config)

			opt.Metrics.JobName = opt.RestoreSessionName
			opt.Host, err = util.GetRestoreHostName(opt.StashClient, opt.RestoreSessionName, opt.Namespace)
			if err != nil {
				return err
			}
			// run restore
			restoreOutput, restoreErr := restore.Restore(opt)
			if restoreErr != nil {
				err = opt.HandleRestoreFailure(restoreErr)
				return errors.NewAggregate([]error{restoreErr, err})

			}
			if restoreOutput != nil {
				return opt.HandleRestoreSuccess(restoreOutput)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opt.MasterURL, "master", opt.MasterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&opt.KubeconfigPath, "kubeconfig", opt.KubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.RestoreSessionName, "restoresession", opt.RestoreSessionName, "Name of the respective RestoreSession object.")
	cmd.Flags().DurationVar(&opt.BackoffMaxWait, "backoff-max-wait", 0, "Maximum wait for initial response from kube apiserver; 0 disables the timeout")
	cmd.Flags().BoolVar(&opt.SetupOpt.EnableCache, "enable-cache", opt.SetupOpt.EnableCache, "Specify whether to enable caching for restic")
	cmd.Flags().IntVar(&opt.SetupOpt.MaxConnections, "max-connections", opt.SetupOpt.MaxConnections, "Specify maximum concurrent connections for GCS, Azure and B2 backend")
	cmd.Flags().StringVar(&opt.SetupOpt.SecretDir, "secret-dir", opt.SetupOpt.SecretDir, "Directory where storage secret has been mounted")

	cmd.Flags().BoolVar(&opt.Metrics.Enabled, "metrics-enabled", opt.Metrics.Enabled, "Specify whether to export Prometheus metrics")
	cmd.Flags().StringVar(&opt.Metrics.PushgatewayURL, "pushgateway-url", opt.Metrics.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")
	cmd.Flags().StringVar(&opt.RestoreModel, "restore-model", opt.RestoreModel, "Specify whether using job or init-container to restore (default init-container)")

	return cmd
}
