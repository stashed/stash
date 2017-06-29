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
	"github.com/appscode/stash/pkg/util"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func NewCmdSchedule(version string) *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		workload string
		opt            scheduler.Options = scheduler.Options{
			Namespace:      namespace(),
			ResticName:     "",
			ScratchDir:     "/tmp",
			PushgatewayURL: "http://stash-operator.kube-system.svc:56789",
			NodeNamePath:   "/etc/nodename",
			PodLabelsPath:  "/etc/labels",
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


			app := strings.SplitN(workload, "/", 2)
			if len(app) != 2 {
				log.Fatalf(`--workload flag must be in the format "Kind/Name", but found %v`, workload)
			}
			opt.AppName = app[1]
			switch app[0] {
			case "Deployment", "deployment":
				opt.AppKind = "Deployment"
				if util.IsPreferredAPIResource(kubeClient, apps.GroupName, opt.AppKind) {
					_, err := kubeClient.AppsV1beta1().Deployments(opt.Namespace).Get(opt.AppName, metav1.GetOptions{})
					if err != nil {
						_, err := kubeClient.ExtensionsV1beta1().Deployments(opt.Namespace).Get(opt.AppName, metav1.GetOptions{})
						if err != nil {
							log.Fatalf(`Unknown Deployment %s@%s`, opt.AppName, opt.Namespace)
						}
					}
				} else {
					_, err := kubeClient.ExtensionsV1beta1().Deployments(opt.Namespace).Get(opt.AppName, metav1.GetOptions{})
					if err != nil {
						log.Fatalf(`Unknown Deployment %s@%s`, opt.AppName, opt.Namespace)
					}
				}
			case "ReplicaSet", "replicaset", "rs":
				opt.AppKind = "ReplicaSet"
				_, err := kubeClient.ExtensionsV1beta1().ReplicaSets(opt.Namespace).Get(opt.AppName, metav1.GetOptions{})
				if err != nil {
					log.Fatalf(`Unknown ReplicaSet %s@%s`, opt.AppName, opt.Namespace)
				}
			case "ReplicationController", "replicationcontroller", "rc":
				opt.AppKind = "ReplicationController"
				_, err := kubeClient.CoreV1().ReplicationControllers(opt.Namespace).Get(opt.AppName, metav1.GetOptions{})
				if err != nil {
					log.Fatalf(`Unknown ReplicationController %s@%s`, opt.AppName, opt.Namespace)
				}
			case "StatefulSet", "ss":
				opt.AppKind = "StatefulSet"
				_, err := kubeClient.AppsV1beta1().StatefulSets(opt.Namespace).Get(opt.AppName, metav1.GetOptions{})
				if err != nil {
					log.Fatalf(`Unknown StatefulSet %s@%s`, opt.AppName, opt.Namespace)
				}
			case "DaemonSet", "daemonset":
				opt.AppKind = "DaemonSet"
				_, err := kubeClient.ExtensionsV1beta1().DaemonSets(opt.Namespace).Get(opt.AppName, metav1.GetOptions{})
				if err != nil {
					log.Fatalf(`Unknown DaemonSet %s@%s`, opt.AppName, opt.Namespace)
				}
			default:
				log.Fatalf(`Unrecognized workload "Kind" %v`, opt.AppKind)
			}

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
	cmd.Flags().StringVar(&workload, "workload", workload, `"Kind/Name" of workload where sidecar pod is added (eg, Deployment/apiserver)`)
	cmd.Flags().StringVar(&opt.ResticName, "restic-name", opt.ResticName, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.ScratchDir, "scratch-dir", opt.ScratchDir, "Directory used to store temporary files. Use an `emptyDir` in Kubernetes.")
	cmd.Flags().StringVar(&opt.PushgatewayURL, "pushgateway-url", opt.PushgatewayURL, "URL of Prometheus pushgateway used to cache backup metrics")

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
