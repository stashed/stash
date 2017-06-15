package cmd

import (
	"github.com/appscode/log"
	_ "github.com/appscode/restik/api/install"
	rcs "github.com/appscode/restik/client/clientset"
	"github.com/appscode/restik/pkg/controller"
	"github.com/spf13/cobra"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdWatch() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Run restic backup",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get kubernetes config: %s", err)
			}
			kubeClient := clientset.NewForConfigOrDie(config)
			restikClient := rcs.NewForConfigOrDie(config)
			cronController, err := controller.NewCronController(kubeClient, restikClient)
			if err != nil {
				log.Fatalln(err)
			}
			err = cronController.RunBackup()
			if err != nil {
				log.Errorln(err)
			}
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	return cmd
}
