package clientcmd

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func BuildConfigFromContext(kubeconfigPath, contextName string) (*rest.Config, error) {
	var loader clientcmd.ClientConfigLoader
	if kubeconfigPath == "" {
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
		loader = rules
	} else {
		loader = &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	}
	overrides := &clientcmd.ConfigOverrides{
		CurrentContext:  contextName,
		ClusterDefaults: clientcmd.ClusterDefaults,
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides).ClientConfig()
}

func ClientFromContext(kubeconfigPath, contextName string) (kubernetes.Interface, error) {
	cfg, err := BuildConfigFromContext(kubeconfigPath, contextName)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

func NamespaceFromContext(kubeconfigPath, contextName string) (string, error) {
	kConfig, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return "", err
	}
	ctx, found := kConfig.Contexts[contextName]
	if !found {
		return "", fmt.Errorf("context %s not found in kubeconfig file %s", contextName, kubeconfigPath)
	}
	return ctx.Namespace, nil
}
