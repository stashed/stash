package cmds

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/log"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	cs "github.com/appscode/stash/client/clientset/versioned"
	stash_scheme "github.com/appscode/stash/client/clientset/versioned/scheme"
	v1beta1_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"github.com/appscode/stash/pkg/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/reference"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/meta"
)

type options struct {
	name        string
	namespace   string
	k8sClient   kubernetes.Interface
	stashClient cs.Interface
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
		Name:            rand.WithUniqSuffix(opt.name),
		Namespace:       opt.namespace,
		OwnerReferences: []metav1.OwnerReference{},
	}
	backupConfiguration, err := opt.stashClient.StashV1beta1().BackupConfigurations(opt.namespace).Get(opt.name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	// skip if BackupConfiguration paused
	if backupConfiguration.Spec.Paused {
		log.Infof("Skipping creating BackupSession. Reason: Backup Configuration %s/%s is paused.", backupConfiguration.Namespace, backupConfiguration.Name)
		return nil
	}
	ref, err := reference.GetReference(stash_scheme.Scheme, backupConfiguration)
	if err != nil {
		return err
	}
	_, _, err = v1beta1_util.CreateOrPatchBackupSession(opt.stashClient.StashV1beta1(), bsMeta, func(in *api_v1beta1.BackupSession) *api_v1beta1.BackupSession {
		// Set Backupconfiguration  as Backupsession Owner
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
