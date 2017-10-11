/*
Copyright 2017 The Stash Authors.

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

package internalversion

import (
	"github.com/appscode/stash/client/internalclientset/scheme"
	rest "k8s.io/client-go/rest"
)

type StashInterface interface {
	RESTClient() rest.Interface
	RecoveriesGetter
	ResticsGetter
}

// StashClient is used to interact with features provided by the stash.appscode.com group.
type StashClient struct {
	restClient rest.Interface
}

func (c *StashClient) Recoveries(namespace string) RecoveryInterface {
	return newRecoveries(c, namespace)
}

func (c *StashClient) Restics(namespace string) ResticInterface {
	return newRestics(c, namespace)
}

// NewForConfig creates a new StashClient for the given config.
func NewForConfig(c *rest.Config) (*StashClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &StashClient{client}, nil
}

// NewForConfigOrDie creates a new StashClient for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *StashClient {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new StashClient for the given RESTClient.
func New(c rest.Interface) *StashClient {
	return &StashClient{c}
}

func setConfigDefaults(config *rest.Config) error {
	g, err := scheme.Registry.Group("stash.appscode.com")
	if err != nil {
		return err
	}

	config.APIPath = "/apis"
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	if config.GroupVersion == nil || config.GroupVersion.Group != g.GroupVersion.Group {
		gv := g.GroupVersion
		config.GroupVersion = &gv
	}
	config.NegotiatedSerializer = scheme.Codecs

	if config.QPS == 0 {
		config.QPS = 5
	}
	if config.Burst == 0 {
		config.Burst = 10
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *StashClient) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
