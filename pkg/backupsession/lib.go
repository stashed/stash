package backupsession

import (
	"github.com/appscode/go/crypto/rand"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	cs "github.com/appscode/stash/client/clientset/versioned"
	"github.com/appscode/stash/client/clientset/versioned/scheme"
	stash_v1beta1_util "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	"github.com/appscode/stash/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/reference"
	core_util "kmodules.xyz/client-go/core/v1"
)

type Controller struct {
	Options
	k8sClient   kubernetes.Interface
	stashClient cs.Interface
}

type Options struct {
	Name      string
	Namespace string
}

func New(k8sClient kubernetes.Interface, stashClient cs.Interface, opt Options) *Controller {
	return &Controller{
		k8sClient:   k8sClient,
		stashClient: stashClient,
		Options:     opt,
	}
}

func (c *Controller) CreateBackupSession() error {
	meta := metav1.ObjectMeta{
		Name:            rand.WithUniqSuffix(c.Name),
		Namespace:       c.Namespace,
		OwnerReferences: []metav1.OwnerReference{},
	}
	backupConfiguration, err := c.stashClient.StashV1beta1().BackupConfigurations(c.Namespace).Get(c.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	ref, err := reference.GetReference(scheme.Scheme, backupConfiguration)
	if err != nil {
		return err
	}
	_, _, err = stash_v1beta1_util.CreateOrPatchBackupSession(c.stashClient.StashV1beta1(), meta, func(in *api_v1beta1.BackupSession) *api_v1beta1.BackupSession {
		//Set Backupconfiguration  as Backupsession Owner
		core_util.EnsureOwnerReference(&in.ObjectMeta, ref)
		in.Spec.BackupConfiguration.Name = c.Name
		if in.Labels == nil {
			in.Labels = map[string]string{}
		}
		in.Labels[util.LabelApp] = util.AppLabelStash
		in.Labels[util.LabelBackupConfiguration] = backupConfiguration.Name
		return in
	})
	return err
}
