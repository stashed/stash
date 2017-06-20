package fake

import (
	rapi "github.com/appscode/stash/api"
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

var stashResource = schema.GroupVersionResource{Group: "backup.appscode.com", Version: "v1alpha1", Resource: "stashs"}

var _ clientset.StashInterface = &FakeStash{}

// Get returns the Stashs by name.
func (mock *FakeStash) Get(name string) (*rapi.Restic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewGetAction(stashResource, mock.ns, name), &rapi.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*rapi.Restic), err
}

// List returns the a of Stashs.
func (mock *FakeStash) List(opts metav1.ListOptions) (*rapi.ResticList, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewListAction(stashResource, mock.ns, opts), &rapi.Restic{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &rapi.ResticList{}
	for _, item := range obj.(*rapi.ResticList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Create creates a new Stash.
func (mock *FakeStash) Create(svc *rapi.Restic) (*rapi.Restic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewCreateAction(stashResource, mock.ns, svc), &rapi.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*rapi.Restic), err
}

// Update updates a Stash.
func (mock *FakeStash) Update(svc *rapi.Restic) (*rapi.Restic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateAction(stashResource, mock.ns, svc), &rapi.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*rapi.Restic), err
}

// Delete deletes a Stash by name.
func (mock *FakeStash) Delete(name string, _ *metav1.DeleteOptions) error {
	_, err := mock.Fake.
		Invokes(testing.NewDeleteAction(stashResource, mock.ns, name), &rapi.Restic{})

	return err
}

func (mock *FakeStash) UpdateStatus(srv *rapi.Restic) (*rapi.Restic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateSubresourceAction(stashResource, "status", mock.ns, srv), &rapi.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*rapi.Restic), err
}

func (mock *FakeStash) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return mock.Fake.
		InvokesWatch(testing.NewWatchAction(stashResource, mock.ns, opts))
}
