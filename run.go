package main

import (
	"fmt"
	"net/http"

	stringz "github.com/appscode/go/strings"
	"github.com/appscode/log"
	"github.com/appscode/pat"
	sapi "github.com/appscode/stash/api"
	scs "github.com/appscode/stash/client/clientset"
	"github.com/appscode/stash/pkg/analytics"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/pkg/docker"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeClient  clientset.Interface
	stashClient scs.ExtensionInterface

	scratchDir string = "/tmp"
)

func NewCmdRun(version string) *cobra.Command {
	var (
		masterURL       string
		kubeconfigPath  string
		tag             string = stringz.Val(Version, "canary")
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
			kubeClient = clientset.NewForConfigOrDie(config)
			stashClient = scs.NewForConfigOrDie(config)

			ctrl := controller.New(kubeClient, stashClient, tag)
			err = ctrl.Setup()
			if err != nil {
				log.Fatalln(err)
			}

			log.Infoln("Starting operator...")
			ctrl.Run()

			m := pat.New()
			m.Get("/metrics", promhttp.Handler())

			pattern := fmt.Sprintf("/%s/v1beta1/namespaces/%s/restics/%s/metrics", sapi.GroupName, PathParamNamespace, PathParamName)
			log.Infof("URL pattern: %s", pattern)
			m.Get(pattern, http.HandlerFunc(ExportSnapshots))

			http.Handle("/", m)
			log.Infoln("Listening on", address)
			log.Fatal(http.ListenAndServe(address, nil))
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&address, "address", address, "Address to listen on for web interface and telemetry.")
	cmd.Flags().StringVar(&scratchDir, "scratch-dir", scratchDir, "Directory used to store temporary files. Use an `emptyDir` in Kubernetes.")
	cmd.Flags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical event to Google Analytics")

	return cmd
}
