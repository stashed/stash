package framework

import (
	"path/filepath"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/kutil/tools/certstore"
	cs "github.com/appscode/stash/client/clientset/versioned"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"k8s.io/client-go/kubernetes"
	//ka "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

type Framework struct {
	KubeClient     kubernetes.Interface
	StashClient    cs.Interface
	//KAClient       ka.Clientset
	namespace      string
	CertStore      *certstore.CertStore
	WebhookEnabled bool
}

func New(kubeClient kubernetes.Interface, extClient cs.Interface, webhookEnabled bool) *Framework {
	store, err := certstore.NewCertStore(afero.NewMemMapFs(), filepath.Join("", "pki"))
	Expect(err).NotTo(HaveOccurred())

	err = store.InitCA()
	Expect(err).NotTo(HaveOccurred())

	return &Framework{
		KubeClient:     kubeClient,
		StashClient:    extClient,
		//KAClient:       kaClient,
		namespace:      rand.WithUniqSuffix("test-stash"),
		CertStore:      store,
		WebhookEnabled: webhookEnabled,
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
