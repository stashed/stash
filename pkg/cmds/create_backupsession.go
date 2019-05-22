package cmds

import (
	"fmt"
	"strings"
	"time"

	"github.com/appscode/go/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/reference"
	"k8s.io/kubernetes/pkg/apis/core"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/discovery"
	"kmodules.xyz/client-go/meta"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
	oc_cs "kmodules.xyz/openshift/client/clientset/versioned"
	"stash.appscode.dev/stash/apis"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	stash_scheme "stash.appscode.dev/stash/client/clientset/versioned/scheme"
	v1beta1_util "stash.appscode.dev/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/util"
)

type options struct {
	name             string
	namespace        string
	k8sClient        kubernetes.Interface
	stashClient      cs.Interface
	appcatalogClient appcatalog_cs.Interface
	ocClient         oc_cs.Interface
}

func NewCmdCreateBackupSession() *cobra.Command {
	var (
		masterURL      string
		kubeconfigPath string

		opt = options{
			namespace: meta.Namespace(),
		}
	)

	cmd := &cobra.Command{
		Use:               "create-backupsession",
		Short:             "create a BackupSession",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
			if err != nil {
				log.Fatalf("Could not get Kubernetes config: %s", err)
			}
			opt.k8sClient = kubernetes.NewForConfigOrDie(config)
			opt.stashClient = cs.NewForConfigOrDie(config)
			opt.appcatalogClient = appcatalog_cs.NewForConfigOrDie(config)
			// if cluster has OpenShift DeploymentConfig then generate OcClient
			if discovery.IsPreferredAPIResource(opt.k8sClient.Discovery(), ocapps.GroupVersion.String(), apis.KindDeploymentConfig) {
				opt.ocClient = oc_cs.NewForConfigOrDie(config)
			}

			err = opt.createBackupSession()
			if err != nil {
				log.Fatal(err)
			}

		},
	}

	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	cmd.Flags().StringVar(&opt.name, "backupsession.name", "", "Set BackupSession Name")
	cmd.Flags().StringVar(&opt.namespace, "backupsession.namespace", opt.namespace, "Set BackupSession Namespace")

	return cmd
}

func (opt *options) createBackupSession() error {
	bsMeta := metav1.ObjectMeta{
		// Name format: <BackupConfiguration name>-<timestamp in unix format>
		Name:            fmt.Sprintf("%s-%d", opt.name, time.Now().Unix()),
		Namespace:       opt.namespace,
		OwnerReferences: []metav1.OwnerReference{},
	}
	backupConfiguration, err := opt.stashClient.StashV1beta1().BackupConfigurations(opt.namespace).Get(opt.name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	// skip if BackupConfiguration paused
	if backupConfiguration.Spec.Paused {
		msg := fmt.Sprintf("Skipping creating BackupSession. Reason: Backup Configuration %s/%s is paused.", backupConfiguration.Namespace, backupConfiguration.Name)
		log.Infoln(msg)

		// write event to BackupConfiguration denoting that backup session has been skipped
		return writeBackupSessionSkippedEvent(opt.k8sClient, backupConfiguration, msg)
	}
	// if target does not exist then skip creating BackupSession
	wc := util.WorkloadClients{
		KubeClient:       opt.k8sClient,
		StashClient:      opt.stashClient,
		AppCatalogClient: opt.appcatalogClient,
		OcClient:         opt.ocClient,
	}

	if backupConfiguration.Spec.Target != nil && !wc.IsTargetExist(backupConfiguration.Spec.Target.Ref, backupConfiguration.Namespace) {
		msg := fmt.Sprintf("Skipping creating BackupSession. Reason: Target workload %s/%s does not exist.",
			strings.ToLower(backupConfiguration.Spec.Target.Ref.Kind), backupConfiguration.Spec.Target.Ref.Name)
		log.Infoln(msg)

		// write event to BackupConfiguration denoting that backup session has been skipped
		return writeBackupSessionSkippedEvent(opt.k8sClient, backupConfiguration, msg)
	}

	// create BackupSession
	ref, err := reference.GetReference(stash_scheme.Scheme, backupConfiguration)
	if err != nil {
		return err
	}
	_, _, err = v1beta1_util.CreateOrPatchBackupSession(opt.stashClient.StashV1beta1(), bsMeta, func(in *api_v1beta1.BackupSession) *api_v1beta1.BackupSession {
		// Set BackupConfiguration  as BackupSession Owner
		core_util.EnsureOwnerReference(&in.ObjectMeta, ref)
		in.Spec.BackupConfiguration.Name = opt.name
		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels[util.LabelApp] = util.AppLabelStash
		in.Labels[util.LabelBackupConfiguration] = backupConfiguration.Name
		return in
	})
	return err
}

func writeBackupSessionSkippedEvent(kubeClient kubernetes.Interface, backupConfiguration *api_v1beta1.BackupConfiguration, msg string) error {
	_, err := eventer.CreateEvent(
		kubeClient,
		eventer.EventSourceBackupTriggeringCronJob,
		backupConfiguration,
		core.EventTypeNormal,
		eventer.EventReasonBackupSkipped,
		msg,
	)
	return err
}
