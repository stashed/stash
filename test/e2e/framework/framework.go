package framework

import (
	"github.com/appscode/go/crypto/rand"
	scs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	clientset "k8s.io/client-go/kubernetes"
)

type Framework struct {
	KubeClient  clientset.Interface
	StashClient scs.ResticsGetter
	namespace   string
}

func New(kubeClient clientset.Interface, extClient scs.ResticsGetter) *Framework {
	return &Framework{
		KubeClient:  kubeClient,
		StashClient: extClient,
		namespace:   rand.WithUniqSuffix("test-stash"),
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
