package cmd

import (
	"fmt"
	"time"

	_ "github.com/appscode/restik/api/install"
	"github.com/appscode/restik/pkg/controller"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/util/runtime"
)

func NewCmdRun() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run restic operator",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				fmt.Printf("Could not get kubernetes config: %s", err)
				time.Sleep(30 * time.Minute)
				panic(err)
			}
			defer runtime.HandleCrash()

			w := controller.New(config)
			fmt.Println("Starting tillerc...")
			w.RunAndHold()
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	return cmd
}
