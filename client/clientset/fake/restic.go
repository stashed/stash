package fake

import (
	tapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/client/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"
)

type FakeStash struct {
	Fake *testing.Fake
	ns   string
}

var stashResource = tapi.V1alpha1SchemeGroupVersion.WithResource(tapi.ResourceTypeRestic)

var _ clientset.ResticInterface = &FakeStash{}

// Get returns the Stashs by name.
func (mock *FakeStash) Get(name string) (*tapi.Restic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewGetAction(stashResource, mock.ns, name), &tapi.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*tapi.Restic), err
}

// List returns the a of Stashs.
func (mock *FakeStash) List(opts metav1.ListOptions) (*tapi.ResticList, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewListAction(stashResource, mock.ns, opts), &tapi.Restic{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &tapi.ResticList{}
	for _, item := range obj.(*tapi.ResticList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Create creates a new Stash.
func (mock *FakeStash) Create(svc *tapi.Restic) (*tapi.Restic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewCreateAction(stashResource, mock.ns, svc), &tapi.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*tapi.Restic), err
}

// Update updates a Stash.
func (mock *FakeStash) Update(svc *tapi.Restic) (*tapi.Restic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateAction(stashResource, mock.ns, svc), &tapi.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*tapi.Restic), err
}

// Delete deletes a Stash by name.
func (mock *FakeStash) Delete(name string, _ *metav1.DeleteOptions) error {
	_, err := mock.Fake.
		Invokes(testing.NewDeleteAction(stashResource, mock.ns, name), &tapi.Restic{})

	return err
}

func (mock *FakeStash) UpdateStatus(srv *tapi.Restic) (*tapi.Restic, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateSubresourceAction(stashResource, "status", mock.ns, srv), &tapi.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*tapi.Restic), err
}

func (mock *FakeStash) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return mock.Fake.
		InvokesWatch(testing.NewWatchAction(stashResource, mock.ns, opts))
}
