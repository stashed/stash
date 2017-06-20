package main

import (
	"net/http"

	"github.com/appscode/log"
	rcs "github.com/appscode/restik/client/clientset"
	"github.com/appscode/restik/pkg/analytics"
	"github.com/appscode/restik/pkg/controller"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdRun(version string) *cobra.Command {
	var (
		masterURL       string
		kubeconfigPath  string
		image           string
		address         string = ":56790"
		enableAnalytics bool   = true
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run restic operator",
		PreRun: func(cmd *cobra.Command, args []string) {
			if enableAnalytics {
				analytics.Enable()
			}
			analytics.SendEvent("operator", "started", version)
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			analytics.SendEvent("operator", "stopped", version)
		},
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalln(err)
			}
			kubeClient := clientset.NewForConfigOrDie(config)
			restikClient := rcs.NewForConfigOrDie(config)

			ctrl := controller.NewRestikController(kubeClient, restikClient, image)
			err = ctrl.Setup()
			if err != nil {
				log.Fatalln(err)
			}

			log.Infoln("Starting operator...")
			go ctrl.RunAndHold()

			http.Handle("/metrics", promhttp.Handler())
			log.Infoln("Listening on", address)
			log.Fatal(http.ListenAndServe(address, nil))
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&image, "image", "appscode/restik:latest", "Image that will be used by restic-sidecar container.")
	cmd.Flags().StringVar(&address, "address", address, "Address to listen on for web interface and telemetry.")
	cmd.Flags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical event to Google Analytics")

	return cmd
}
