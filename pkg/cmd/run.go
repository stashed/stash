package cmd

import (
	"fmt"
	"time"

	_ "github.com/appscode/k8s-addons/api/install"
	"github.com/appscode/restik/pkg/controller"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/util/runtime"
)

func NewCmdRun() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		image          string
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

			w := controller.New(config, image)
			fmt.Println("Starting restik controller...")
			w.RunAndHold()
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&image, "image", "appscode/restik:latest", "Image that will be used by restic-sidecar container.")

	return cmd
}
