package clientset

import (
	tapi "github.com/appscode/stash/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type ResticGetter interface {
	Restics(namespace string) ResticInterface
}

type ResticInterface interface {
	List(opts metav1.ListOptions) (*tapi.ResticList, error)
	Get(name string) (*tapi.Restic, error)
	Create(stash *tapi.Restic) (*tapi.Restic, error)
	Update(stash *tapi.Restic) (*tapi.Restic, error)
	Delete(name string, options *metav1.DeleteOptions) error
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	UpdateStatus(stash *tapi.Restic) (*tapi.Restic, error)
}

type ResticImpl struct {
	r  rest.Interface
	ns string
}

var _ ResticInterface = &ResticImpl{}

func newRestic(c *ExtensionClient, namespace string) *ResticImpl {
	return &ResticImpl{c.restClient, namespace}
}

func (c *ResticImpl) List(opts metav1.ListOptions) (result *tapi.ResticList, err error) {
	result = &tapi.ResticList{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource(tapi.ResourceTypeRestic).
		VersionedParams(&opts, ExtendedCodec).
		Do().
		Into(result)
	return
}

func (c *ResticImpl) Get(name string) (result *tapi.Restic, err error) {
	result = &tapi.Restic{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource(tapi.ResourceTypeRestic).
		Name(name).
		Do().
		Into(result)
	return
}

func (c *ResticImpl) Create(stash *tapi.Restic) (result *tapi.Restic, err error) {
	result = &tapi.Restic{}
	err = c.r.Post().
		Namespace(c.ns).
		Resource(tapi.ResourceTypeRestic).
		Body(stash).
		Do().
		Into(result)
	return
}

func (c *ResticImpl) Update(stash *tapi.Restic) (result *tapi.Restic, err error) {
	result = &tapi.Restic{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource(tapi.ResourceTypeRestic).
		Name(stash.Name).
		Body(stash).
		Do().
		Into(result)
	return
}

func (c *ResticImpl) Delete(name string, options *metav1.DeleteOptions) (err error) {
	return c.r.Delete().
		Namespace(c.ns).
		Resource(tapi.ResourceTypeRestic).
		Name(name).
		Body(options).
		Do().
		Error()
}

func (c *ResticImpl) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return c.r.Get().
		Prefix("watch").
		Namespace(c.ns).
		Resource(tapi.ResourceTypeRestic).
		VersionedParams(&opts, ExtendedCodec).
		Watch()
}

func (c *ResticImpl) UpdateStatus(stash *tapi.Restic) (result *tapi.Restic, err error) {
	result = &tapi.Restic{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource(tapi.ResourceTypeRestic).
		Name(stash.Name).
		SubResource("status").
		Body(stash).
		Do().
		Into(result)
	return
}
