package framework

import (
	"path/filepath"

	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"gomodules.xyz/cert/certstore"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ka "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
)

type Framework struct {
	KubeClient   kubernetes.Interface
	StashClient  cs.Interface
	KAClient     ka.Interface
	namespace    string
	CertStore    *certstore.CertStore
	ClientConfig *rest.Config
	StorageClass string
}

func New(kubeClient kubernetes.Interface, extClient cs.Interface, kaClient ka.Interface, clientConfig *rest.Config, storageClass string) *Framework {
	store, err := certstore.NewCertStore(afero.NewMemMapFs(), filepath.Join("", "pki"))
	Expect(err).NotTo(HaveOccurred())

	err = store.InitCA()
	Expect(err).NotTo(HaveOccurred())

	return &Framework{
		KubeClient:   kubeClient,
		StashClient:  extClient,
		KAClient:     kaClient,
		namespace:    rand.WithUniqSuffix("test-stash"),
		CertStore:    store,
		ClientConfig: clientConfig,
		StorageClass: storageClass,
	}
}

func (f *Framework) Invoke() *Invocation {
	return &Invocation{
		Framework:       f,
		app:             rand.WithUniqSuffix("stash-e2e"),
		BackupBlueprint: &v1beta1.BackupBlueprint{},
		BackupSession:   &v1beta1.BackupSession{},
		Repository:      &v1alpha1.Repository{},
		BackupConfig:    &v1beta1.BackupConfiguration{},
	}
}

func (f *Invocation) AppLabel() string {
	return "app=" + f.app
}

func (f *Invocation) App() string {
	return f.app
}

type Invocation struct {
	*Framework
	app             string
	BackupConfig    *v1beta1.BackupConfiguration
	BackupSession   *v1beta1.BackupSession
	BackupBlueprint *v1beta1.BackupBlueprint
	Repository      *v1alpha1.Repository
}
