package fake

import (
	rapi "github.com/appscode/restik/api"
	"github.com/appscode/restik/client/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"
)

type FakeRestik struct {
	Fake *testing.Fake
	ns   string
}

var restikResource = schema.GroupVersionResource{Group: "backup.appscode.com", Version: "v1alpha1", Resource: "restiks"}

var _ clientset.RestikInterface = &FakeRestik{}

// Get returns the Restiks by name.
func (mock *FakeRestik) Get(name string) (*rapi.Restik, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewGetAction(restikResource, mock.ns, name), &rapi.Restik{})

	if obj == nil {
		return nil, err
	}
	return obj.(*rapi.Restik), err
}

// List returns the a of Restiks.
func (mock *FakeRestik) List(opts metav1.ListOptions) (*rapi.RestikList, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewListAction(restikResource, mock.ns, opts), &rapi.Restik{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &rapi.RestikList{}
	for _, item := range obj.(*rapi.RestikList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Create creates a new Restik.
func (mock *FakeRestik) Create(svc *rapi.Restik) (*rapi.Restik, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewCreateAction(restikResource, mock.ns, svc), &rapi.Restik{})

	if obj == nil {
		return nil, err
	}
	return obj.(*rapi.Restik), err
}

// Update updates a Restik.
func (mock *FakeRestik) Update(svc *rapi.Restik) (*rapi.Restik, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateAction(restikResource, mock.ns, svc), &rapi.Restik{})

	if obj == nil {
		return nil, err
	}
	return obj.(*rapi.Restik), err
}

// Delete deletes a Restik by name.
func (mock *FakeRestik) Delete(name string, _ *metav1.DeleteOptions) error {
	_, err := mock.Fake.
		Invokes(testing.NewDeleteAction(restikResource, mock.ns, name), &rapi.Restik{})

	return err
}

func (mock *FakeRestik) UpdateStatus(srv *rapi.Restik) (*rapi.Restik, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateSubresourceAction(restikResource, "status", mock.ns, srv), &rapi.Restik{})

	if obj == nil {
		return nil, err
	}
	return obj.(*rapi.Restik), err
}

func (mock *FakeRestik) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return mock.Fake.
		InvokesWatch(testing.NewWatchAction(restikResource, mock.ns, opts))
}
