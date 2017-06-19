package main

import (
	"github.com/appscode/log"
	rcs "github.com/appscode/restik/client/clientset"
	"github.com/appscode/restik/pkg/analytics"
	"github.com/appscode/restik/pkg/controller"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdRun(version string) *cobra.Command {
	var (
		masterURL       string
		kubeconfigPath  string
		image           string
		enableAnalytics bool = true
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run restic operator",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get kubernetes config: %s", err)
			}
			kubeClient := clientset.NewForConfigOrDie(config)
			restikClient := rcs.NewForConfigOrDie(config)
			ctrl := controller.NewRestikController(kubeClient, restikClient, image)

			log.Infoln("Starting restik operator...")

			if enableAnalytics {
				analytics.Enable()
			}
			analytics.SendEvent(image, "started", version)

			defer runtime.HandleCrash()
			err = ctrl.RunAndHold()
			if err != nil {
				log.Errorln(err)
			}
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&image, "image", "appscode/restik:latest", "Image that will be used by restic-sidecar container.")

	// Analytics flags
	cmd.Flags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical event to Google Analytics")

	return cmd
}
