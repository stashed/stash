package clientset

import (
	"fmt"

	schema "k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apimachinery/registered"
	rest "k8s.io/kubernetes/pkg/client/restclient"
)

const (
	defaultAPIPath = "/apis"
)

type AppsCodeRestikInterface interface {
	RESTClient() rest.Interface
	RestikNamespacer
}

// AppsCodeRestikClient is used to interact with experimental Kubernetes features.
// Features of Extensions group are not supported and may be changed or removed in
// incompatible ways at any time.
type AppsCodeRestikClient struct {
	restClient rest.Interface
}

func (a *AppsCodeRestikClient) Restiks(namespace string) RestikInterface {
	return newRestik(a, namespace)
}

// NewACRestikForConfig creates a new AppsCodeRestikClient for the given config. This client
// provides access to experimental Kubernetes features.
// Features of Extensions group are not supported and may be changed or removed in
// incompatible ways at any time.
func NewACRestikForConfig(c *rest.Config) (*AppsCodeRestikClient, error) {
	config := *c
	if err := setRestikDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &AppsCodeRestikClient{client}, nil
}

// NewACRestikForConfigOrDie creates a new AppsCodeRestikClient for the given config and
// panics if there is an error in the config.
// Features of Extensions group are not supported and may be changed or removed in
// incompatible ways at any time.
func NewACRestikForConfigOrDie(c *rest.Config) *AppsCodeRestikClient {
	client, err := NewACRestikForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new ExtensionsV1beta1Client for the given RESTClient.
func NewACRestik(c rest.Interface) *AppsCodeRestikClient {
	return &AppsCodeRestikClient{c}
}

func setRestikDefaults(config *rest.Config) error {
	gv, err := schema.ParseGroupVersion("backup.appscode.com/v1beta1")
	if err != nil {
		return err
	}
	// if backup.appscode.com/v1beta1 is not enabled, return an error
	if !registered.IsEnabledVersion(gv) {
		return fmt.Errorf("backup.appscode.com/v1beta1 is not enabled")
	}
	config.APIPath = defaultAPIPath
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	if config.GroupVersion == nil || config.GroupVersion.Group != "backup.appscode.com" {
		g, err := registered.Group("backup.appscode.com")
		if err != nil {
			return err
		}
		copyGroupVersion := g.GroupVersion
		config.GroupVersion = &copyGroupVersion
	}

	config.NegotiatedSerializer = DirectCodecFactory{extendedCodec: ExtendedCodec}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *AppsCodeRestikClient) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
