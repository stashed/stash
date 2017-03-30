package clientset

import (
	aci "github.com/appscode/k8s-addons/api"
	"k8s.io/kubernetes/pkg/api"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/watch"
)

type AlertNamespacer interface {
	Alert(namespace string) AlertInterface
}

type AlertInterface interface {
	List(opts api.ListOptions) (*aci.AlertList, error)
	Get(name string) (*aci.Alert, error)
	Create(Alert *aci.Alert) (*aci.Alert, error)
	Update(Alert *aci.Alert) (*aci.Alert, error)
	Delete(name string) error
	Watch(opts api.ListOptions) (watch.Interface, error)
	UpdateStatus(Alert *aci.Alert) (*aci.Alert, error)
}

type AlertImpl struct {
	r  rest.Interface
	ns string
}

func newAlert(c *AppsCodeExtensionsClient, namespace string) *AlertImpl {
	return &AlertImpl{c.restClient, namespace}
}

func (c *AlertImpl) List(opts api.ListOptions) (result *aci.AlertList, err error) {
	result = &aci.AlertList{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource("alerts").
		VersionedParams(&opts, ExtendedCodec).
		Do().
		Into(result)
	return
}

func (c *AlertImpl) Get(name string) (result *aci.Alert, err error) {
	result = &aci.Alert{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource("alerts").
		Name(name).
		Do().
		Into(result)
	return
}

func (c *AlertImpl) Create(alert *aci.Alert) (result *aci.Alert, err error) {
	result = &aci.Alert{}
	err = c.r.Post().
		Namespace(c.ns).
		Resource("alerts").
		Body(alert).
		Do().
		Into(result)
	return
}

func (c *AlertImpl) Update(alert *aci.Alert) (result *aci.Alert, err error) {
	result = &aci.Alert{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource("alerts").
		Name(alert.Name).
		Body(alert).
		Do().
		Into(result)
	return
}

func (c *AlertImpl) Delete(name string) (err error) {
	return c.r.Delete().
		Namespace(c.ns).
		Resource("alerts").
		Name(name).
		Do().
		Error()
}

func (c *AlertImpl) Watch(opts api.ListOptions) (watch.Interface, error) {
	return c.r.Get().
		Prefix("watch").
		Namespace(c.ns).
		Resource("alerts").
		VersionedParams(&opts, ExtendedCodec).
		Watch()
}

func (c *AlertImpl) UpdateStatus(alert *aci.Alert) (result *aci.Alert, err error) {
	result = &aci.Alert{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource("alerts").
		Name(alert.Name).
		SubResource("status").
		Body(alert).
		Do().
		Into(result)
	return
}
