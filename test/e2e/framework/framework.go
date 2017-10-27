package framework

import (
	"github.com/appscode/go/crypto/rand"
	cs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	"k8s.io/client-go/kubernetes"
)

type Framework struct {
	KubeClient  kubernetes.Interface
	StashClient cs.StashV1alpha1Interface
	namespace   string
}

func New(kubeClient kubernetes.Interface, extClient cs.StashV1alpha1Interface) *Framework {
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
