package clientset

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/rest"
)

const (
	defaultAPIPath = "/apis"
)

type ExtensionInterface interface {
	RESTClient() rest.Interface
	RestikNamespacer
}

// ExtensionsClient is used to interact with experimental Kubernetes features.
// Features of Extensions group are not supported and may be changed or removed in
// incompatible ways at any time.
type ExtensionClient struct {
	restClient rest.Interface
}

var _ ExtensionInterface = &ExtensionClient{}

func (a *ExtensionClient) Restiks(namespace string) RestikInterface {
	return newRestik(a, namespace)
}

// NewForConfig creates a new ExtensionClient for the given config. This client
// provides access to experimental Kubernetes features.
// Features of Extensions group are not supported and may be changed or removed in
// incompatible ways at any time.
func NewForConfig(c *rest.Config) (*ExtensionClient, error) {
	config := *c
	if err := setExtensionsDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &ExtensionClient{client}, nil
}

// NewForConfigOrDie creates a new ExtensionClient for the given config and
// panics if there is an error in the config.
// Features of Extensions group are not supported and may be changed or removed in
// incompatible ways at any time.
func NewForConfigOrDie(c *rest.Config) *ExtensionClient {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new ExtensionClient for the given RESTClient.
func New(c rest.Interface) *ExtensionClient {
	return &ExtensionClient{c}
}

func setExtensionsDefaults(config *rest.Config) error {
	gv, err := schema.ParseGroupVersion("backup.appscode.com/v1alpha1")
	if err != nil {
		return err
	}
	// if backup.appscode.com/v1alpha1 is not enabled, return an error
	if !api.Registry.IsEnabledVersion(gv) {
		return fmt.Errorf("backup.appscode.com/v1alpha1 is not enabled")
	}
	config.APIPath = defaultAPIPath
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	if config.GroupVersion == nil || config.GroupVersion.Group != "backup.appscode.com" {
		g, err := api.Registry.Group("backup.appscode.com")
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
func (c *ExtensionClient) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
