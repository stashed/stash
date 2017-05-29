package fake

import (
	aci "github.com/appscode/k8s-addons/api"
	"k8s.io/kubernetes/pkg/api"
	schema "k8s.io/kubernetes/pkg/api/unversioned"
	testing "k8s.io/kubernetes/pkg/client/testing/core"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
)

type FakeCertificate struct {
	Fake *testing.Fake
	ns   string
}

var certResource = schema.GroupVersionResource{Group: "appscode.com", Version: "v1beta1", Resource: "certificates"}

// Get returns the Certificate by name.
func (mock *FakeCertificate) Get(name string) (*aci.Certificate, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewGetAction(certResource, mock.ns, name), &aci.Certificate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Certificate), err
}

// List returns the a of Certificates.
func (mock *FakeCertificate) List(opts api.ListOptions) (*aci.CertificateList, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewListAction(certResource, mock.ns, opts), &aci.Certificate{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &aci.CertificateList{}
	for _, item := range obj.(*aci.CertificateList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Create creates a new Certificate.
func (mock *FakeCertificate) Create(svc *aci.Certificate) (*aci.Certificate, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewCreateAction(certResource, mock.ns, svc), &aci.Certificate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Certificate), err
}

// Update updates a Certificate.
func (mock *FakeCertificate) Update(svc *aci.Certificate) (*aci.Certificate, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateAction(certResource, mock.ns, svc), &aci.Certificate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Certificate), err
}

// Delete deletes a Certificate by name.
func (mock *FakeCertificate) Delete(name string) error {
	_, err := mock.Fake.
		Invokes(testing.NewDeleteAction(certResource, mock.ns, name), &aci.Certificate{})

	return err
}

func (mock *FakeCertificate) UpdateStatus(srv *aci.Certificate) (*aci.Certificate, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateSubresourceAction(certResource, "status", mock.ns, srv), &aci.Certificate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Certificate), err
}

func (mock *FakeCertificate) Watch(opts api.ListOptions) (watch.Interface, error) {
	return mock.Fake.
		InvokesWatch(testing.NewWatchAction(certResource, mock.ns, opts))
}
