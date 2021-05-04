/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clientcmd

import (
	"net"
	"os"

	"kmodules.xyz/client-go/meta"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

// BuildConfigFromFlags is a helper function that builds configs from a master
// url or a kubeconfig filepath. These are passed in as command line flags for cluster
// components. Warnings should reflect this usage. If neither masterUrl or kubeconfigPath
// are passed in we fallback to inClusterConfig. If inClusterConfig fails, we fallback
// to the default config.
func BuildConfigFromFlags(masterUrl, kubeconfigPath string) (*rest.Config, error) {
	return fix(clientcmd.BuildConfigFromFlags(masterUrl, kubeconfigPath))
}

// BuildConfigFromKubeconfigGetter is a helper function that builds configs from a master
// url and a kubeconfigGetter.
func BuildConfigFromKubeconfigGetter(masterUrl string, kubeconfigGetter clientcmd.KubeconfigGetter) (*rest.Config, error) {
	return fix(clientcmd.BuildConfigFromKubeconfigGetter(masterUrl, kubeconfigGetter))
}

func BuildConfigFromContext(kubeconfigPath, contextName string) (*rest.Config, error) {
	var loader clientcmd.ClientConfigLoader
	if kubeconfigPath == "" {
		if meta.PossiblyInCluster() {
			return rest.InClusterConfig()
		}
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
		loader = rules
	} else {
		loader = &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	}
	overrides := &clientcmd.ConfigOverrides{
		CurrentContext: contextName,
	}
	return fix(clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides).ClientConfig())
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
		return "", errors.Errorf("context %s not found in kubeconfig file %s", contextName, kubeconfigPath)
	}
	return ctx.Namespace, nil
}

func fix(cfg *rest.Config, err error) (*rest.Config, error) {
	return Fix(cfg), err
}

var fixAKS = true

func init() {
	pflag.BoolVar(&fixAKS, "use-kubeapiserver-fqdn-for-aks", fixAKS, "if true, uses kube-apiserver FQDN for AKS cluster to workaround https://github.com/Azure/AKS/issues/522")
}

func UseKubeAPIServerFQDNForAKS() bool {
	return fixAKS
}

// FixAKS uses kube-apiserver FQDN for AKS cluster to workaround https://github.com/Azure/AKS/issues/522
func Fix(cfg *rest.Config) *rest.Config {
	if cfg == nil || !fixAKS {
		return cfg
	}

	// ref: https://github.com/kubernetes/client-go/blob/kubernetes-1.11.3/rest/config.go#L309
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) > 0 &&
		len(port) > 0 &&
		in(cfg.Host, "https://"+net.JoinHostPort(host, port), "https://kubernetes.default.svc", "https://kubernetes.default.svc:443") {
		// uses service ip or cluster dns

		if cert, err := meta.APIServerCertificate(cfg); err == nil {
			// kube-apiserver cert found

			if host, err := meta.TestAKS(cert); err == nil {
				// AKS cluster

				h := "https://" + host
				klog.Infof("resetting Kubeconfig host to %s from %s for AKS to workaround https://github.com/Azure/AKS/issues/522", h, cfg.Host)
				cfg.Host = h
			}
		}
	}
	return cfg
}

func in(x string, a ...string) bool {
	for _, v := range a {
		if x == v {
			return true
		}
	}
	return false
}
