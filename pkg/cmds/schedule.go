package cmds

import (
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/kutil"
	cs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	"github.com/appscode/stash/pkg/scheduler"
	"github.com/appscode/stash/pkg/util"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdSchedule() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            = scheduler.Options{
			Namespace:      kutil.Namespace(),
			ResticName:     "",
			ScratchDir:     "/tmp",
			PushgatewayURL: "http://stash-operator.kube-system.svc:56789",
			PodLabelsPath:  "/etc/stash/labels",
			ResyncPeriod:   5 * time.Minute,
			MaxNumRequeues: 5,
		}
	)

	cmd := &cobra.Command{
		Use:               "schedule",
		Short:             "Run Stash cron daemon",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get Kubernetes config: %s", err)
			}
			kubeClient = kubernetes.NewForConfigOrDie(config)
			stashClient = cs.NewForConfigOrDie(config)

			opt.NodeName = os.Getenv("NODE_NAME")
			if opt.NodeName == "" {
				log.Fatalln(`Missing ENV var "NODE_NAME"`)
			}
			opt.PodName = os.Getenv("POD_NAME")
			if opt.PodName == "" {
				log.Fatalln(`Missing ENV var "POD_NAME"`)
			}

			if err := opt.Workload.Canonicalize(); err != nil {
				log.Fatalf(err.Error())
			}
			if opt.SnapshotHostname, opt.SmartPrefix, err = opt.Workload.HostnamePrefix(opt.PodName, opt.NodeName); err != nil {
				log.Fatalf(err.Error())
			}
			if err = util.CheckWorkloadExists(kubeClient, opt.Namespace, opt.Workload); err != nil {
				log.Fatalf(err.Error())
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

			ctrl := scheduler.New(kubeClient, stashClient, opt)
			err = ctrl.Setup()
			if err != nil {
				log.Fatalf("Failed to setup scheduler: %s", err)
			}
			stop := make(chan struct{})
			defer close(stop)
			go ctrl.Run(1, stop)

			// Wait forever
			select {}
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.Workload.Kind, "workload-kind", opt.Workload.Kind, "Kind of workload where sidecar pod is added.")
	cmd.Flags().StringVar(&opt.Workload.Name, "workload-name", opt.Workload.Name, "Name of workload where sidecar pod is added.")
	cmd.Flags().StringVar(&opt.ResticName, "restic-name", opt.ResticName, "Name of the Restic used as configuration.")
	cmd.Flags().StringVar(&opt.ScratchDir, "scratch-dir", opt.ScratchDir, "Directory used to store temporary files. Use an `emptyDir` in Kubernetes.")
	cmd.Flags().StringVar(&opt.PushgatewayURL, "pushgateway-url", opt.PushgatewayURL, "URL of Prometheus pushgateway used to cache backup metrics")
	cmd.Flags().DurationVar(&opt.ResyncPeriod, "resync-period", opt.ResyncPeriod, "If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out.")

	return cmd
}
