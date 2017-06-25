package framework

import (
	scs "github.com/appscode/stash/client/clientset"
	clientset "k8s.io/client-go/kubernetes"
)

type Framework struct {
	kubeClient  clientset.Interface
	stashClient scs.ExtensionInterface
}

func New(kubeClient clientset.Interface, extClient scs.ExtensionInterface) *Framework {
	return &Framework{
		kubeClient:  kubeClient,
		stashClient: extClient,
	}
}
