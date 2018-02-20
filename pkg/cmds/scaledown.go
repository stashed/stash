package cmds

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kutil/meta"
	"github.com/appscode/stash/pkg/scale"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdScaleDown() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            = scale.Options{
			Namespace: meta.Namespace(),
		}
	)

	cmd := &cobra.Command{
		Use:               "scaledown",
		Short:             "Scale down workload",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get Kubernetes config: %s", err)
			}
			kubeClient := kubernetes.NewForConfigOrDie(config)

			ctrl := scale.New(kubeClient, opt)
			err = ctrl.ScaleDownWorkload()
			if err != nil {
				log.Fatal(err)
			}

		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.Selector, "selector", opt.Selector, "Label used to select Restic's workload")

	return cmd
}
