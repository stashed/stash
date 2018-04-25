package v1

import (
	occ "github.com/openshift/client-go/apps/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Interface interface {
	WorkloadsGetter
}

// Client is used to interact with features provided by the storage.k8s.io group.
type Client struct {
	kc kubernetes.Interface
	oc occ.Interface
}

func (c *Client) Workloads(namespace string) WorkloadInterface {
	return newWorkloads(c.kc, c.oc, namespace)
}

// NewForConfig creates a new Client for the given config.
func NewForConfig(c *rest.Config) (*Client, error) {
	kc, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err
	}
	oc, err := occ.NewForConfig(c)
	if err != nil {
		return nil, err
	}
	return &Client{kc, oc}, nil
}

// NewForConfigOrDie creates a new Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Client {
	return &Client{kubernetes.NewForConfigOrDie(c), occ.NewForConfigOrDie(c)}
}

// New creates a new Client for the given RESTClient.
func New(kc kubernetes.Interface, oc occ.Interface) *Client {
	return &Client{kc, oc}
}
