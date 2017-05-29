package clientset

import (
	aci "github.com/appscode/k8s-addons/api"
	"k8s.io/kubernetes/pkg/api"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/watch"
)

type CertificateNamespacer interface {
	Certificate(namespace string) CertificateInterface
}

type CertificateInterface interface {
	List(opts api.ListOptions) (*aci.CertificateList, error)
	Get(name string) (*aci.Certificate, error)
	Create(certificate *aci.Certificate) (*aci.Certificate, error)
	Update(certificate *aci.Certificate) (*aci.Certificate, error)
	Delete(name string) error
	Watch(opts api.ListOptions) (watch.Interface, error)
	UpdateStatus(certificate *aci.Certificate) (*aci.Certificate, error)
}

type CertificateImpl struct {
	r  rest.Interface
	ns string
}

func newCertificate(c *AppsCodeExtensionsClient, namespace string) *CertificateImpl {
	return &CertificateImpl{c.restClient, namespace}
}

func (c *CertificateImpl) List(opts api.ListOptions) (result *aci.CertificateList, err error) {
	result = &aci.CertificateList{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource("certificates").
		VersionedParams(&opts, ExtendedCodec).
		Do().
		Into(result)
	return
}

func (c *CertificateImpl) Get(name string) (result *aci.Certificate, err error) {
	result = &aci.Certificate{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource("certificates").
		Name(name).
		Do().
		Into(result)
	return
}

func (c *CertificateImpl) Create(certificate *aci.Certificate) (result *aci.Certificate, err error) {
	result = &aci.Certificate{}
	err = c.r.Post().
		Namespace(c.ns).
		Resource("certificates").
		Body(certificate).
		Do().
		Into(result)
	return
}

func (c *CertificateImpl) Update(certificate *aci.Certificate) (result *aci.Certificate, err error) {
	result = &aci.Certificate{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource("certificates").
		Name(certificate.Name).
		Body(certificate).
		Do().
		Into(result)
	return
}

func (c *CertificateImpl) Delete(name string) (err error) {
	return c.r.Delete().
		Namespace(c.ns).
		Resource("certificates").
		Name(name).
		Do().
		Error()
}

func (c *CertificateImpl) Watch(opts api.ListOptions) (watch.Interface, error) {
	return c.r.Get().
		Prefix("watch").
		Namespace(c.ns).
		Resource("certificates").
		VersionedParams(&opts, ExtendedCodec).
		Watch()
}

func (c *CertificateImpl) UpdateStatus(certificate *aci.Certificate) (result *aci.Certificate, err error) {
	result = &aci.Certificate{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource("certificates").
		Name(certificate.Name).
		SubResource("status").
		Body(certificate).
		Do().
		Into(result)
	return
}
