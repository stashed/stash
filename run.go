package main

import (
	"net/http"

	stringz "github.com/appscode/go/strings"
	v "github.com/appscode/go/version"
	"github.com/appscode/log"
	rcs "github.com/appscode/stash/client/clientset"
	"github.com/appscode/stash/pkg/analytics"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/pkg/docker"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdRun(version string) *cobra.Command {
	var (
		masterURL       string
		kubeconfigPath  string
		tag             string = stringz.Val(v.Version.Version, "canary")
		address         string = ":56790"
		enableAnalytics bool   = true
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run Stash operator",
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
			if err := docker.CheckDockerImageVersion(docker.ImageOperator, tag); err != nil {
				log.Fatalf(`Image %v:%v not found.`, docker.ImageOperator, tag)
			}

			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalln(err)
			}
			kubeClient := clientset.NewForConfigOrDie(config)
			stashClient := rcs.NewForConfigOrDie(config)

			ctrl := controller.NewController(kubeClient, stashClient, tag)
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
	cmd.Flags().StringVar(&address, "address", address, "Address to listen on for web interface and telemetry.")
	cmd.Flags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical event to Google Analytics")

	return cmd
}
