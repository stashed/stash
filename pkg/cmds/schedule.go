package cmds

import (
	"io/ioutil"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		masterURL      string
		kubeconfigPath string
		opt            scheduler.Options = scheduler.Options{
			ResticNamespace: namespace(),
			ResticName:      "",
			PrefixHostname:  true,
			ScratchDir:      "/tmp",
			PushgatewayURL:  "http://stash-operator.kube-system.svc:56789",
			PodLabelsPath:   "/etc/labels",
		}
		enableAnalytics bool = true
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
			kubeClient = clientset.NewForConfigOrDie(config)
			stashClient = rcs.NewForConfigOrDie(config)

			opt.ScratchDir = strings.TrimSuffix(opt.ScratchDir, "/")
			err = os.MkdirAll(opt.ScratchDir, 0755)
			if err != nil {
				log.Fatalf("Failed to create scratch dir: %s", err)
			}
			err = ioutil.WriteFile(opt.ScratchDir+"/.stash", []byte("test"), 644)
			if err != nil {
				log.Fatalf("No write access in scratch dir: %s", err)
			}

			ctrl, err := scheduler.New(kubeClient, stashClient, opt)
			if err != nil {
				log.Fatalf("Failed to create scheduler: %s", err)
			}
			err = ctrl.Setup()
			if err != nil {
				log.Fatalf("Failed to setup scheduler: %s", err)
			}
			ctrl.RunAndHold()
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.App, "app", opt.App, "Name of app where sidecar pod is added")
	cmd.Flags().StringVar(&opt.ResticName, "restic-name", opt.ResticName, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().BoolVar(&opt.PrefixHostname, "prefix-hostname", opt.PrefixHostname, "If set, adds Hostname as prefix to repository. This should be true for StatefulSets & DaemonSets. This should be false in all other cases.")
	cmd.Flags().StringVar(&opt.ScratchDir, "scratch-dir", opt.ScratchDir, "Directory used to store temporary files. Use an `emptyDir` in Kubernetes.")
	cmd.Flags().StringVar(&opt.PushgatewayURL, "pushgateway-url", opt.PushgatewayURL, "URL of Prometheus pushgateway used to cache backup metrics")
	cmd.Flags().StringVar(&opt.PodLabelsPath, "pod-labels-path", opt.PodLabelsPath, "Path to pod labels file mounted via Kubernetes Downward api")

	// Analytics flags
	cmd.Flags().BoolVar(&enableAnalytics, "analytics", enableAnalytics, "Send analytical events to Google Analytics")
	return cmd
}


func namespace() string {
	if ns := os.Getenv("OPERATOR_NAMESPACE"); ns != "" {
		return ns
	}
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return metav1.NamespaceDefault
}
