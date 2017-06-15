package clientset

import (
	rapi "github.com/appscode/restik/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
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
	List(opts metav1.ListOptions) (*rapi.RestikList, error)
	Get(name string) (*rapi.Restik, error)
	Create(restik *rapi.Restik) (*rapi.Restik, error)
	Update(restik *rapi.Restik) (*rapi.Restik, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	UpdateStatus(restik *rapi.Restik) (*rapi.Restik, error)
}

type RestikImpl struct {
	r  rest.Interface
	ns string
}

var _ RestikInterface = &RestikImpl{}

func newRestik(c *ExtensionClient, namespace string) *RestikImpl {
	return &RestikImpl{c.restClient, namespace}
}

func (c *RestikImpl) List(opts metav1.ListOptions) (result *rapi.RestikList, err error) {
	result = &rapi.RestikList{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource(ResourceTypeRestik).
		VersionedParams(&opts, ExtendedCodec).
		Do().
		Into(result)
	return
}

func (c *RestikImpl) Get(name string) (result *rapi.Restik, err error) {
	result = &rapi.Restik{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource(ResourceTypeRestik).
		Name(name).
		Do().
		Into(result)
	return
}

func (c *RestikImpl) Create(restik *rapi.Restik) (result *rapi.Restik, err error) {
	result = &rapi.Restik{}
	err = c.r.Post().
		Namespace(c.ns).
		Resource(ResourceTypeRestik).
		Body(restik).
		Do().
		Into(result)
	return
}

func (c *RestikImpl) Update(restik *rapi.Restik) (result *rapi.Restik, err error) {
	result = &rapi.Restik{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource(ResourceTypeRestik).
		Name(restik.Name).
		Body(restik).
		Do().
		Into(result)
	return
}

func (c *RestikImpl) Delete(name string, options *metav1.DeleteOptions) (err error) {
	return c.r.Delete().
		Namespace(c.ns).
		Resource(ResourceTypeRestik).
		Name(name).
		Body(options).
		Do().
		Error()
}

func (c *RestikImpl) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return c.r.Get().
		Prefix("watch").
		Namespace(c.ns).
		Resource(ResourceTypeRestik).
		VersionedParams(&opts, ExtendedCodec).
		Watch()
}

func (c *RestikImpl) UpdateStatus(restik *rapi.Restik) (result *rapi.Restik, err error) {
	result = &rapi.Restik{}
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
