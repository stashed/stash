package cmds

import (
	"os"
	"strings"
	"time"

	"github.com/appscode/go/log"
	"github.com/appscode/kutil/meta"
	cs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	"github.com/appscode/stash/pkg/backup"
	"github.com/appscode/stash/pkg/util"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdBackup() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            = backup.Options{
			Namespace:      meta.Namespace(),
			ScratchDir:     "/tmp",
			PodLabelsPath:  "/etc/stash/labels",
			ResyncPeriod:   5 * time.Minute,
			MaxNumRequeues: 5,
		}
	)

	cmd := &cobra.Command{
		Use:               "backup",
		Short:             "Run Stash Backup",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get Kubernetes config: %s", err)
			}
			kubeClient := kubernetes.NewForConfigOrDie(config)
			stashClient := cs.NewForConfigOrDie(config)

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
			if err = util.WorkloadExists(kubeClient, opt.Namespace, opt.Workload); err != nil {
				log.Fatalf(err.Error())
			}
			opt.ScratchDir = strings.TrimSuffix(opt.ScratchDir, "/") // make ScratchDir in setup()

			ctrl := backup.New(kubeClient, stashClient, opt)

			if opt.RunViaCron {
				log.Infoln("Running backup periodically via cron")
				if err = ctrl.BackupScheduler(); err != nil {
					log.Fatal(err)
				}
			} else {
				log.Infoln("Running backup once")
				if err = ctrl.Backup(); err != nil {
					log.Fatal(err)
				}
			}
			log.Infoln("Exiting Stash Backup")
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
	cmd.Flags().BoolVar(&opt.RunViaCron, "run-via-cron", opt.RunViaCron, "Run backup periodically via cron.")
	cmd.Flags().StringVar(&opt.ImageTag, "image-tag", opt.ImageTag, "Check job image tag.")
	cmd.Flags().BoolVar(&opt.EnableRBAC, "enable-rbac", opt.EnableRBAC, "Enable RBAC")

	return cmd
}
