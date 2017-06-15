package fake

import (
	aci "github.com/appscode/restik/api"
	"github.com/appscode/restik/client/clientset"
apiv1 "k8s.io/client-go/pkg/api/v1"
metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
"k8s.io/client-go/testing"
"k8s.io/apimachinery/pkg/labels"
"k8s.io/apimachinery/pkg/watch"
)

type FakeRestik struct {
	Fake *testing.Fake
	ns   string
}

var restikResource = metav1.GroupVersionResource{Group: "backup.appscode.com", Version: "v1alpha1", Resource: "restiks"}

var _ clientset.RestikInterface = &FakeRestik{}

// Get returns the Restiks by name.
func (mock *FakeRestik) Get(name string) (*aci.Restik, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewGetAction(restikResource, mock.ns, name), &aci.Restik{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Restik), err
}

// List returns the a of Restiks.
func (mock *FakeRestik) List(opts apiv1.ListOptions) (*aci.RestikList, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewListAction(restikResource, mock.ns, opts), &aci.Restik{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &aci.RestikList{}
	for _, item := range obj.(*aci.RestikList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Create creates a new Restik.
func (mock *FakeRestik) Create(svc *aci.Restik) (*aci.Restik, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewCreateAction(restikResource, mock.ns, svc), &aci.Restik{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Restik), err
}

// Update updates a Restik.
func (mock *FakeRestik) Update(svc *aci.Restik) (*aci.Restik, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateAction(restikResource, mock.ns, svc), &aci.Restik{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Restik), err
}

// Delete deletes a Restik by name.
func (mock *FakeRestik) Delete(name string, _ *apiv1.DeleteOptions) error {
	_, err := mock.Fake.
		Invokes(testing.NewDeleteAction(restikResource, mock.ns, name), &aci.Restik{})

	return err
}

func (mock *FakeRestik) UpdateStatus(srv *aci.Restik) (*aci.Restik, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateSubresourceAction(restikResource, "status", mock.ns, srv), &aci.Restik{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Restik), err
}

func (mock *FakeRestik) Watch(opts apiv1.ListOptions) (watch.Interface, error) {
	return mock.Fake.
		InvokesWatch(testing.NewWatchAction(restikResource, mock.ns, opts))
}
