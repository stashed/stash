package backup_session

import (
	"github.com/appscode/go/crypto/rand"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/appscode/stash/client/clientset/versioned/scheme"
	cs "github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1"
	"github.com/appscode/stash/client/clientset/versioned/typed/stash/v1beta1/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/reference"
	core_util "kmodules.xyz/client-go/core/v1"
)

type Controller struct {
	Options
	k8sClient          kubernetes.Interface
	stashv1beta1Client cs.StashV1beta1Interface
}

type Options struct {
	Name      string
	Namespace string
}

func New(k8sClient kubernetes.Interface, stashv1betaClient cs.StashV1beta1Interface, opt Options) *Controller {
	return &Controller{
		k8sClient:          k8sClient,
		stashv1beta1Client: stashv1betaClient,
		Options:            opt,
	}
}

func (c *Controller) CreateBackupSession() error {
	meta := metav1.ObjectMeta{
		Name:            rand.WithUniqSuffix(c.Name),
		Namespace:       c.Namespace,
		OwnerReferences: []metav1.OwnerReference{},
	}
	backupConfiguration, err := c.stashv1beta1Client.BackupConfigurations(c.Namespace).Get(c.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	ref, err := reference.GetReference(scheme.Scheme, backupConfiguration)
	if err != nil {
		return err
	}
	_, _, err = util.CreateOrPatchBackupSession(c.stashv1beta1Client, meta, func(in *api_v1beta1.BackupSession) *api_v1beta1.BackupSession {
		//Set Backupconfiguration  as Backupsession Owner
		core_util.EnsureOwnerReference(&in.ObjectMeta, ref)
		in.Spec.BackupConfiguration.Name = c.Name
		return in
	})
	if err != nil {
		return err
	}
	return nil
}
