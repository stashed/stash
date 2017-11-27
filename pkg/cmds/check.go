package cmds

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kutil"
	"github.com/appscode/stash/client/typed/stash/v1alpha1"
	"github.com/appscode/stash/pkg/check"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdCheck() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		resticName     string
		hostName       string
		smartPrefix    string
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
			c := check.New(
				kubernetes.NewForConfigOrDie(config),
				v1alpha1.NewForConfigOrDie(config),
				kutil.Namespace(),
				resticName,
				hostName,
				smartPrefix,
			)
			if err = c.Run(); err != nil {
				log.Fatal(err)
			}
			log.Infoln("Exiting stash check")
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&resticName, "restic-name", resticName, "Name of the Restic CRD.")
	cmd.Flags().StringVar(&hostName, "host-name", hostName, "Host name for workload.")
	cmd.Flags().StringVar(&smartPrefix, "smart-prefix", smartPrefix, "Smart prefix for workload")

	return cmd
}
