package framework

import (
	scs "github.com/appscode/stash/client/clientset"
	clientset "k8s.io/client-go/kubernetes"
)

type Framework struct {
	KubeClient  clientset.Interface
	StashClient scs.ExtensionInterface
}
