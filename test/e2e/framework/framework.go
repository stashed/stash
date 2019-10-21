package framework

import (
	"path/filepath"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"gomodules.xyz/cert/certstore"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ka "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
)

type Framework struct {
	KubeClient   kubernetes.Interface
	StashClient  cs.Interface
	KAClient     ka.Interface
	dmClient     dynamic.Interface
	namespace    string
	CertStore    *certstore.CertStore
	ClientConfig *rest.Config
	StorageClass string
}

// RootFramework will be used to invoke new Invocation before each test from the individual test packages
var RootFramework *Framework

func New(kubeClient kubernetes.Interface, extClient cs.Interface, kaClient ka.Interface, dmClient dynamic.Interface, clientConfig *rest.Config, storageClass string) *Framework {
	store, err := certstore.NewCertStore(afero.NewMemMapFs(), filepath.Join("", "pki"))
	Expect(err).NotTo(HaveOccurred())

	err = store.InitCA()
	Expect(err).NotTo(HaveOccurred())

	return &Framework{
		KubeClient:   kubeClient,
		StashClient:  extClient,
		KAClient:     kaClient,
		dmClient:     dmClient,
		namespace:    rand.WithUniqSuffix("test-stash"),
		CertStore:    store,
		ClientConfig: clientConfig,
		StorageClass: storageClass,
	}
}
func NewInvocation() *Invocation {
	return RootFramework.Invoke()
}

func (f *Framework) Invoke() *Invocation {
	return &Invocation{
		Framework:     f,
		app:           rand.WithUniqSuffix("stash-e2e"),
		testResources: make([]interface{}, 0),
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
	app           string
	testResources []interface{}
}
