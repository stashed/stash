package cmds

import (
	"github.com/appscode/go/log"
	"github.com/appscode/kutil/meta"
	"github.com/appscode/stash/client/typed/stash/v1alpha1"
	"github.com/appscode/stash/pkg/recovery"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdRecover() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		recoveryName   string
	)

	cmd := &cobra.Command{
		Use:               "recover",
		Short:             "Recover restic backup",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalln(err)
			}
			c := recovery.New(
				kubernetes.NewForConfigOrDie(config),
				v1alpha1.NewForConfigOrDie(config),
				meta.Namespace(),
				recoveryName,
			)
			c.Run()
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&recoveryName, "recovery-name", recoveryName, "Name of the Recovery CRD.")

	return cmd
}
