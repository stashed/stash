package clientset

import (
	aci "github.com/appscode/restik/api"
apiv1 "k8s.io/client-go/pkg/api/v1"
"k8s.io/client-go/rest"
"k8s.io/apimachinery/pkg/watch"
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
	List(opts apiv1.ListOptions) (*aci.RestikList, error)
	Get(name string) (*aci.Restik, error)
	Create(restik *aci.Restik) (*aci.Restik, error)
	Update(restik *aci.Restik) (*aci.Restik, error)
	Delete(name string, options *apiv1.DeleteOptions) error
	Watch(opts apiv1.ListOptions) (watch.Interface, error)
	UpdateStatus(restik *aci.Restik) (*aci.Restik, error)
}

type RestikImpl struct {
	r  rest.Interface
	ns string
}

var _ RestikInterface = &RestikImpl{}

func newRestik(c *ExtensionClient, namespace string) *RestikImpl {
	return &RestikImpl{c.restClient, namespace}
}

func (c *RestikImpl) List(opts apiv1.ListOptions) (result *aci.RestikList, err error) {
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

func (c *RestikImpl) Delete(name string, options *apiv1.DeleteOptions) (err error) {
	return c.r.Delete().
		Namespace(c.ns).
		Resource(ResourceTypeRestik).
		Name(name).
		Body(options).
		Do().
		Error()
}

func (c *RestikImpl) Watch(opts apiv1.ListOptions) (watch.Interface, error) {
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
