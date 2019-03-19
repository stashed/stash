package clientcmd

import (
	"os"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type restClientGetter struct {
	config *clientcmdapi.Config
}

var _ genericclioptions.RESTClientGetter = restClientGetter{}

func (r restClientGetter) ToRESTConfig() (*rest.Config, error) {
	return r.ToRawKubeConfigLoader().ClientConfig()
}

func (r restClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	config, err := r.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return discovery.NewCachedDiscoveryClientForConfig(config, os.TempDir(), "", 10*time.Minute)
}

func (r restClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	client, err := r.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(client), nil
}

func (r restClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return clientcmd.NewDefaultClientConfig(*r.config, &clientcmd.ConfigOverrides{})
}

func NewClientGetter(config *clientcmdapi.Config) genericclioptions.RESTClientGetter {
	return &restClientGetter{config}
}

func NewClientGetterFromFlags(fs *pflag.FlagSet) genericclioptions.RESTClientGetter {
	client := genericclioptions.NewConfigFlags()
	if fs != nil {
		client.AddFlags(fs)
	}
	return client
}
