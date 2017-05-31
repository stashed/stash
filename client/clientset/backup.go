package clientset

import (
	aci "github.com/appscode/restik/api"
	"k8s.io/kubernetes/pkg/api"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/watch"
)

type RestikNamespacer interface {
	Restiks(namespace string) RestikInterface
}

const (
	ResourceKindRestik = "Restik"
	ResourceNameRestik = "restik"
	ResourceTypeRestik = "restiks"
)

type RestikInterface interface {
	List(opts api.ListOptions) (*aci.RestikList, error)
	Get(name string) (*aci.Restik, error)
	Create(restik *aci.Restik) (*aci.Restik, error)
	Update(restik *aci.Restik) (*aci.Restik, error)
	Delete(name string, options *api.DeleteOptions) error
	Watch(opts api.ListOptions) (watch.Interface, error)
	UpdateStatus(restik *aci.Restik) (*aci.Restik, error)
}

type RestikImpl struct {
	r  rest.Interface
	ns string
}

func newRestik(c *AppsCodeExtensionsClient, namespace string) *RestikImpl {
	return &RestikImpl{c.restClient, namespace}
}

func (c *RestikImpl) List(opts api.ListOptions) (result *aci.RestikList, err error) {
	result = &aci.RestikList{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource(ResourceTypeRestik).
		VersionedParams(&opts, ExtendedCodec).
		Do().
		Into(result)
	return
}

func (c *RestikImpl) Get(name string) (result *aci.Restik, err error) {
	result = &aci.Restik{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource(ResourceTypeRestik).
		Name(name).
		Do().
		Into(result)
	return
}

func (c *RestikImpl) Create(restik *aci.Restik) (result *aci.Restik, err error) {
	result = &aci.Restik{}
	err = c.r.Post().
		Namespace(c.ns).
		Resource(ResourceTypeRestik).
		Body(restik).
		Do().
		Into(result)
	return
}

func (c *RestikImpl) Update(restik *aci.Restik) (result *aci.Restik, err error) {
	result = &aci.Restik{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource(ResourceTypeRestik).
		Name(restik.Name).
		Body(restik).
		Do().
		Into(result)
	return
}

func (c *RestikImpl) Delete(name string, options *api.DeleteOptions) (err error) {
	return c.r.Delete().
		Namespace(c.ns).
		Resource(ResourceTypeRestik).
		Name(name).
		Body(options).
		Do().
		Error()
}

func (c *RestikImpl) Watch(opts api.ListOptions) (watch.Interface, error) {
	return c.r.Get().
		Prefix("watch").
		Namespace(c.ns).
		Resource(ResourceTypeRestik).
		VersionedParams(&opts, ExtendedCodec).
		Watch()
}

func (c *RestikImpl) UpdateStatus(restik *aci.Restik) (result *aci.Restik, err error) {
	result = &aci.Restik{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource(ResourceTypeRestik).
		Name(restik.Name).
		SubResource("status").
		Body(restik).
		Do().
		Into(result)
	return
}
