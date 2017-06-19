package main

import (
	"github.com/appscode/log"
	rcs "github.com/appscode/restik/client/clientset"
	"github.com/appscode/restik/pkg/analytics"
	"github.com/appscode/restik/pkg/controller"
	"github.com/spf13/cobra"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdWatch(version string) *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string

		enableAnalytics bool = true
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

			analytics.SendEvent("RestikCron", "created", version)
			err = cronController.RunBackup()
			if err != nil {
				log.Errorln(err)
			}
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	// Analytics flags
	cmd.Flags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical event to Google Analytics")
	return cmd
}
