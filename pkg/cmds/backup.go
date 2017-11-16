package cmds

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"

	"github.com/appscode/go/log"
	"github.com/appscode/kutil"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	cs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	"github.com/appscode/stash/pkg/cli"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/spf13/cobra"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	stash_util "github.com/appscode/stash/client/typed/stash/v1alpha1/util"
)

type BackupOptions struct {
	Workload         api.LocalTypedReference
	Namespace        string
	ResticName       string
	ScratchDir       string
	PushgatewayURL   string
	NodeName         string
	PodName          string
	SmartPrefix      string
	SnapshotHostname string
	PodLabelsPath    string

	k8sClient   kubernetes.Interface
	stashClient cs.StashV1alpha1Interface
	resticCLI   *cli.ResticWrapper
	recorder    record.EventRecorder
}

func NewCmdBackup() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string
		opt            = BackupOptions{
			Namespace:      kutil.Namespace(),
			ResticName:     "",
			ScratchDir:     "/tmp",
			PushgatewayURL: "http://stash-operator.kube-system.svc:56789",
			PodLabelsPath:  "/etc/stash/labels",
		}
	)

	cmd := &cobra.Command{
		Use:               "backup",
		Short:             "Run Offline Stash Backup",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get Kubernetes config: %s", err)
			}

			opt.k8sClient = kubernetes.NewForConfigOrDie(config)
			opt.stashClient = cs.NewForConfigOrDie(config)

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
			//if err = util.WorkloadExists(kubeClient, opt.Namespace, opt.Workload); err != nil {
			//	log.Fatalf(err.Error())
			//}

			opt.ScratchDir = strings.TrimSuffix(opt.ScratchDir, "/") // setup ScratchDir in SetupAndRun
			err = os.MkdirAll(opt.ScratchDir, 0755)
			if err != nil {
				log.Fatalf("Failed to create scratch dir: %s", err)
			}
			err = ioutil.WriteFile(opt.ScratchDir+"/.stash", []byte("test"), 644)
			if err != nil {
				log.Fatalf("No write access in scratch dir: %s", err)
			}

			opt.recorder = eventer.NewEventRecorder(opt.k8sClient, "stash-backup")
			opt.resticCLI = cli.New(opt.ScratchDir, opt.SnapshotHostname)

			if err = opt.runBackup(); err != nil {
				log.Fatalln(err)
			}
		},
	}
	cmd.Flags().StringVar(&masterURL, "master", masterURL, "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", kubeconfigPath, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.Workload.Kind, "workload-kind", opt.Workload.Kind, "Kind of workload where sidecar pod is added.")
	cmd.Flags().StringVar(&opt.Workload.Name, "workload-name", opt.Workload.Name, "Name of workload where sidecar pod is added.")
	cmd.Flags().StringVar(&opt.ResticName, "restic-name", opt.ResticName, "Name of the Restic used as configuration.")
	cmd.Flags().StringVar(&opt.ScratchDir, "scratch-dir", opt.ScratchDir, "Directory used to store temporary files. Use an `emptyDir` in Kubernetes.")
	cmd.Flags().StringVar(&opt.PushgatewayURL, "pushgateway-url", opt.PushgatewayURL, "URL of Prometheus pushgateway used to cache backup metrics")

	return cmd
}

func (opt *BackupOptions) runBackup() error {
	resource, err := opt.stashClient.Restics(opt.Namespace).Get(opt.ResticName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	log.Infof("Found restic %s", resource.Name)
	if err := resource.IsValid(); err != nil {
		return err
	}
	if resource.Spec.Backend.StorageSecretName == "" {
		return errors.New("missing repository secret name")
	}
	secret, err := opt.k8sClient.CoreV1().Secrets(resource.Namespace).Get(resource.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	err = opt.resticCLI.SetupEnv(resource, secret, opt.SmartPrefix)
	if err != nil {
		return err
	}
	err = opt.resticCLI.InitRepositoryIfAbsent()
	if err != nil {
		return err
	}

	for _, fg := range resource.Spec.FileGroups {
		err = opt.resticCLI.Backup(resource, fg)
		if err != nil {
			log.Errorln("Backup operation failed for Restic %s/%s due to %s", resource.Namespace, resource.Name, err)
			opt.recorder.Event(resource.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToBackup, " Error taking backup: "+err.Error())
			return err
		} else {
			hostname, _ := os.Hostname()
			opt.recorder.Event(resource.ObjectReference(), core.EventTypeNormal, eventer.EventReasonSuccessfulBackup, "Backed up pod:"+hostname+" path:"+fg.Path)
		}
		err = opt.resticCLI.Forget(resource, fg)
		if err != nil {
			log.Errorln("Failed to forget old snapshots for Restic %s/%s due to %s", resource.Namespace, resource.Name, err)
			opt.recorder.Event(resource.ObjectReference(), core.EventTypeWarning, eventer.EventReasonFailedToRetention, " Error forgetting snapshots: "+err.Error())
			return err
		}
	}

	_, err = stash_util.PatchRestic(opt.stashClient, resource, func(in *api.Restic) *api.Restic {
		in.Status.BackupCount++
		return in
	})

	return err
}
