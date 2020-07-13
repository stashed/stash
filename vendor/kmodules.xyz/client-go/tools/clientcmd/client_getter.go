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
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
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

	// Don't use disk based cache as that makes it unsafe for multi-tenant backend servers
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	return memory.NewMemCacheClient(discoveryClient), nil
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
	client := genericclioptions.NewConfigFlags(true)
	if fs != nil {
		client.AddFlags(fs)
	}
	return client
}
