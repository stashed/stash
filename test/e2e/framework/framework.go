package framework

import (
	"path/filepath"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/kutil/tools/certstore"
	cs "github.com/appscode/stash/client"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"k8s.io/client-go/kubernetes"
)

type Framework struct {
	KubeClient  kubernetes.Interface
	StashClient cs.Interface
	namespace   string
	CertStore   *certstore.CertStore
}

func New(kubeClient kubernetes.Interface, extClient cs.Interface) *Framework {
	store, err := certstore.NewCertStore(afero.NewMemMapFs(), filepath.Join("", "pki"))
	Expect(err).NotTo(HaveOccurred())

	err = store.InitCA()
	Expect(err).NotTo(HaveOccurred())

	return &Framework{
		KubeClient:  kubeClient,
		StashClient: extClient,
		namespace:   rand.WithUniqSuffix("test-stash"),
		CertStore:   store,
	}
}

func (f *Framework) Invoke() *Invocation {
	return &Invocation{
		Framework: f,
		app:       rand.WithUniqSuffix("stash-e2e"),
	}
}

type Invocation struct {
	*Framework
	app string
}
