package fake

import (
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/client/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"
)

type FakeStash struct {
	Fake *testing.Fake
	ns   string
}

var stashResource = schema.GroupVersionResource{Group: sapi.GroupName, Version: "v1alpha1", Resource: "stashs"}

var _ clientset.ResticInterface = &FakeStash{}

// Get returns the Stashs by name.
func (mock *FakeStash) Get(name string) (*sapi.Restic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewGetAction(stashResource, mock.ns, name), &sapi.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*sapi.Restic), err
}

// List returns the a of Stashs.
func (mock *FakeStash) List(opts metav1.ListOptions) (*sapi.ResticList, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewListAction(stashResource, mock.ns, opts), &sapi.Restic{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &sapi.ResticList{}
	for _, item := range obj.(*sapi.ResticList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Create creates a new Stash.
func (mock *FakeStash) Create(svc *sapi.Restic) (*sapi.Restic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewCreateAction(stashResource, mock.ns, svc), &sapi.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*sapi.Restic), err
}

// Update updates a Stash.
func (mock *FakeStash) Update(svc *sapi.Restic) (*sapi.Restic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateAction(stashResource, mock.ns, svc), &sapi.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*sapi.Restic), err
}

// Delete deletes a Stash by name.
func (mock *FakeStash) Delete(name string, _ *metav1.DeleteOptions) error {
	_, err := mock.Fake.
		Invokes(testing.NewDeleteAction(stashResource, mock.ns, name), &sapi.Restic{})

	return err
}

func (mock *FakeStash) UpdateStatus(srv *sapi.Restic) (*sapi.Restic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateSubresourceAction(stashResource, "status", mock.ns, srv), &sapi.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*sapi.Restic), err
}

func (mock *FakeStash) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return mock.Fake.
		InvokesWatch(testing.NewWatchAction(stashResource, mock.ns, opts))
}
