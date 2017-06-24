package main

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/appscode/log"
	rcs "github.com/appscode/stash/client/clientset"
	"github.com/appscode/stash/pkg/analytics"
	"github.com/appscode/stash/pkg/scheduler"
	"github.com/spf13/cobra"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdSchedule(version string) *cobra.Command {
	var (
		masterURL       string
		kubeconfigPath  string
		namespace       string
		name            string
		prefixHostname  bool   = true
		scratchDir      string = "/tmp"
		enableAnalytics bool   = true
	)

	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Run Stash cron daemon",
		PreRun: func(cmd *cobra.Command, args []string) {
			if enableAnalytics {
				analytics.Enable()
			}
			analytics.SendEvent("scheduler", "started", version)
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			analytics.SendEvent("scheduler", "stopped", version)
		},
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get Kubernetes config: %s", err)
			}
			kubeClient := clientset.NewForConfigOrDie(config)
			stashClient := rcs.NewForConfigOrDie(config)

			scratchDir = strings.TrimSuffix(scratchDir, "/")
			err = os.MkdirAll(scratchDir, 0755)
			if err != nil {
				log.Fatalf("Failed to create scratch dir: %s", err)
			}
			err = ioutil.WriteFile(scratchDir+"/.stash", []byte("test"), 644)
			if err != nil {
				log.Fatalf("No write access in scratch dir: %s", err)
			}

			ctrl := scheduler.NewController(kubeClient, stashClient, namespace, name, prefixHostname, scratchDir)
			err = ctrl.Setup()
			if err != nil {
				log.Fatalf("Failed to setup scheduler: %s", err)
			}
			ctrl.RunAndHold()
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&namespace, "namespace", namespace, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&name, "name", name, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().BoolVar(&prefixHostname, "prefix-hostname", prefixHostname, "If set, adds Hostname as prefix to repository. This should be true for StatefulSets & DaemonSets. This should be false in all other cases.")
	cmd.Flags().StringVar(&scratchDir, "scratch-dir", scratchDir, "Directory used to store temporary files. Use an `emptyDir` in Kubernetes.")

	// Analytics flags
	cmd.Flags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical event to Google Analytics")
	return cmd
}
