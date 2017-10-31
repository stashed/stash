package cmds

import (
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/kutil"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	cs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	"github.com/appscode/stash/pkg/scheduler"
	"github.com/appscode/stash/pkg/util"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
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

			stopBackup := make(chan struct{})
			defer close(stopBackup)

			// split code from here for leader election
			switch opt.Workload.Kind {
			case api.AppKindDeployment, api.AppKindReplicaSet, api.AppKindReplicationController:
				electLeader(kubeClient, opt, stopBackup)
			default:
				setupAndRun(kubeClient, opt, stopBackup)
			}

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

func setupAndRun(k8sClient kubernetes.Interface, opt scheduler.Options, stop chan struct{}) {
	var err error
	if opt.SnapshotHostname, opt.SmartPrefix, err = opt.Workload.HostnamePrefixForWorkload(opt.PodName, opt.NodeName); err != nil {
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

	ctrl := scheduler.New(k8sClient, stashClient, opt)
	err = ctrl.Setup()
	if err != nil {
		log.Fatalf("Failed to setup scheduler: %s", err)
	}

	ctrl.Run(1, stop)
}

func electLeader(k8sClient kubernetes.Interface, opt scheduler.Options, stopBackup chan struct{}) {
	configMap := &core.ConfigMap{ // TODO: add owner ref
		ObjectMeta: metav1.ObjectMeta{
			Name:      scheduler.ConfigMapPrefix + opt.Workload.Name,
			Namespace: opt.Namespace,
		},
	}
	if _, err := k8sClient.CoreV1().ConfigMaps(opt.Namespace).Create(configMap); err != nil && !kerr.IsAlreadyExists(err) {
		log.Fatal(err)
	}

	resLock := &resourcelock.ConfigMapLock{
		ConfigMapMeta: configMap.ObjectMeta,
		Client:        k8sClient.CoreV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      opt.PodName,
			EventRecorder: &record.FakeRecorder{}, // TODO: replace with real event
		},
	}

	go func() {
		leaderelection.RunOrDie(leaderelection.LeaderElectionConfig{
			Lock:          resLock,
			LeaseDuration: scheduler.LeaderElectionLease,
			RenewDeadline: scheduler.LeaderElectionLease * 2 / 3,
			RetryPeriod:   scheduler.LeaderElectionLease / 3,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(stop <-chan struct{}) {
					log.Infoln("Got leadership, preparing backup scheduler")
					setupAndRun(k8sClient, opt, stopBackup)
				},
				OnStoppedLeading: func() {
					log.Infoln("Lost leadership, stopping backup scheduler")
					stopBackup <- struct{}{}
				},
			},
		})
	}()
}
