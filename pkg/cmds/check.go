package cmds

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kutil/meta"
	cs "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1alpha1"
	"github.com/appscode/stash/pkg/check"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdCheck() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            = check.Options{
			Namespace: meta.Namespace(),
		}
	)

	cmd := &cobra.Command{
		Use:               "check",
		Short:             "Check restic backup",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalln(err)
			}
			kubeClient := kubernetes.NewForConfigOrDie(config)
			stashClient := cs.NewForConfigOrDie(config)

			c := check.New(kubeClient, stashClient, opt)
			if err = c.Run(); err != nil {
				log.Fatal(err)
			}
			log.Infoln("Exiting stash check")
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.ResticName, "restic-name", opt.ResticName, "Name of the Restic CRD.")
	cmd.Flags().StringVar(&opt.HostName, "host-name", opt.HostName, "Host name for workload.")
	cmd.Flags().StringVar(&opt.SmartPrefix, "smart-prefix", opt.SmartPrefix, "Smart prefix for workload")

	return cmd
}
