package clientset

import (
	sapi "github.com/appscode/stash/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type ResticGetter interface {
	Restics(namespace string) ResticInterface
}

const (
	ResourceKindRestic = "Restic"
	ResourceNameRestic = "restic"
	ResourceTypeRestic = "restics"
)

type ResticInterface interface {
	List(opts metav1.ListOptions) (*sapi.ResticList, error)
	Get(name string) (*sapi.Restic, error)
	Create(stash *sapi.Restic) (*sapi.Restic, error)
	Update(stash *sapi.Restic) (*sapi.Restic, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	UpdateStatus(stash *sapi.Restic) (*sapi.Restic, error)
}

type ResticImpl struct {
	r  rest.Interface
	ns string
}

var _ ResticInterface = &ResticImpl{}

func newRestic(c *ExtensionClient, namespace string) *ResticImpl {
	return &ResticImpl{c.restClient, namespace}
}

func (c *ResticImpl) List(opts metav1.ListOptions) (result *sapi.ResticList, err error) {
	result = &sapi.ResticList{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource(ResourceTypeRestic).
		VersionedParams(&opts, ExtendedCodec).
		Do().
		Into(result)
	return
}

func (c *ResticImpl) Get(name string) (result *sapi.Restic, err error) {
	result = &sapi.Restic{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource(ResourceTypeRestic).
		Name(name).
		Do().
		Into(result)
	return
}

func (c *ResticImpl) Create(stash *sapi.Restic) (result *sapi.Restic, err error) {
	result = &sapi.Restic{}
	err = c.r.Post().
		Namespace(c.ns).
		Resource(ResourceTypeRestic).
		Body(stash).
		Do().
		Into(result)
	return
}

func (c *ResticImpl) Update(stash *sapi.Restic) (result *sapi.Restic, err error) {
	result = &sapi.Restic{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource(ResourceTypeRestic).
		Name(stash.Name).
		Body(stash).
		Do().
		Into(result)
	return
}

func (c *ResticImpl) Delete(name string, options *metav1.DeleteOptions) (err error) {
	return c.r.Delete().
		Namespace(c.ns).
		Resource(ResourceTypeRestic).
		Name(name).
		Body(options).
		Do().
		Error()
}

func (c *ResticImpl) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return c.r.Get().
		Prefix("watch").
		Namespace(c.ns).
		Resource(ResourceTypeRestic).
		VersionedParams(&opts, ExtendedCodec).
		Watch()
}

func (c *ResticImpl) UpdateStatus(stash *sapi.Restic) (result *sapi.Restic, err error) {
	result = &sapi.Restic{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource(ResourceTypeRestic).
		Name(stash.Name).
		SubResource("status").
		Body(stash).
		Do().
		Into(result)
	return
}
