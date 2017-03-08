package backup

import (
	_ "github.com/appscode/restik/api/install"
	"github.com/appscode/restik/pkg/controller"
	"github.com/spf13/cobra"
)

func NewCmdBackup() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
	)

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "",
		Run: func(cmd *cobra.Command, args []string) {

			controller.RunBackup()
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	return cmd
}
