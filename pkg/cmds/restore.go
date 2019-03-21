package cmds

import (
	"github.com/appscode/go/log"
	cs "github.com/appscode/stash/client/clientset/versioned"
	"github.com/appscode/stash/pkg/restic"
	"github.com/appscode/stash/pkg/restore"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"kmodules.xyz/client-go/meta"
)

func NewCmdRestore() *cobra.Command {
	opt := &restore.Options{
		MasterURL:      "",
		KubeconfigPath: "",
		Namespace:      meta.Namespace(),
		SetupOpt: restic.SetupOptions{
			ScratchDir:  "/tmp",
			EnableCache: false,
		},
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
			// run restore
			err = restore.Restore(opt)
			if err != nil {
				// set RestoreSession status "Failed", write event and send prometheus metrics
				e2 := restore.HandleRestoreFailure(opt, err)
				if e2 != nil {
					err = errors.NewAggregate([]error{err, e2})
				}
				log.Fatalln(err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opt.MasterURL, "master", opt.MasterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&opt.KubeconfigPath, "kubeconfig", opt.KubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.RestoreSessionName, "restore-session", opt.RestoreSessionName, "Name of the RestoreSession CRD.")
	cmd.Flags().DurationVar(&opt.BackoffMaxWait, "backoff-max-wait", 0, "Maximum wait for initial response from kube apiserver; 0 disables the timeout")
	cmd.Flags().BoolVar(&opt.SetupOpt.EnableCache, "enable-cache", opt.SetupOpt.EnableCache, "Specify weather to enable caching for restic")
	cmd.Flags().StringVar(&opt.SetupOpt.SecretDir, "secret-dir", opt.SetupOpt.SecretDir, "Directory where storage secret has been mounted")

	cmd.Flags().BoolVar(&opt.Metrics.Enabled, "metrics-enabled", opt.Metrics.Enabled, "Specify weather to export Prometheus metrics")
	cmd.Flags().StringVar(&opt.Metrics.PushgatewayURL, "pushgateway-url", opt.Metrics.PushgatewayURL, "Pushgateway URL where the metrics will be pushed")

	return cmd
}
