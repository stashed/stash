package fake

import (
	aci "github.com/appscode/restik/api"
	"k8s.io/kubernetes/pkg/api"
	schema "k8s.io/kubernetes/pkg/api/unversioned"
	testing "k8s.io/kubernetes/pkg/client/testing/core"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
)

type FakeBackup struct {
	Fake *testing.Fake
	ns   string
}

var certResource = schema.GroupVersionResource{Group: "appscode.com", Version: "v1beta1", Resource: "certificates"}

// Get returns the Certificate by name.
func (mock *FakeBackup) Get(name string) (*aci.Backup, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewGetAction(certResource, mock.ns, name), &aci.Backup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Backup), err
}

// List returns the a of Certificates.
func (mock *FakeBackup) List(opts api.ListOptions) (*aci.BackupList, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewListAction(certResource, mock.ns, opts), &aci.Backup{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &aci.BackupList{}
	for _, item := range obj.(*aci.BackupList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Create creates a new Certificate.
func (mock *FakeBackup) Create(svc *aci.Backup) (*aci.Backup, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewCreateAction(certResource, mock.ns, svc), &aci.Backup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Backup), err
}

// Update updates a Certificate.
func (mock *FakeBackup) Update(svc *aci.Backup) (*aci.Backup, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateAction(certResource, mock.ns, svc), &aci.Backup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Backup), err
}

// Delete deletes a Certificate by name.
func (mock *FakeBackup) Delete(name string, _ *api.DeleteOptions) error {
	_, err := mock.Fake.
		Invokes(testing.NewDeleteAction(certResource, mock.ns, name), &aci.Backup{})

	return err
}

func (mock *FakeBackup) UpdateStatus(srv *aci.Backup) (*aci.Backup, error) {
	obj, err := mock.Fake.
		Invokes(testing.NewUpdateSubresourceAction(certResource, "status", mock.ns, srv), &aci.Backup{})

	if obj == nil {
		return nil, err
	}
	return obj.(*aci.Backup), err
}

func (mock *FakeBackup) Watch(opts api.ListOptions) (watch.Interface, error) {
	return mock.Fake.
		InvokesWatch(testing.NewWatchAction(certResource, mock.ns, opts))
}
