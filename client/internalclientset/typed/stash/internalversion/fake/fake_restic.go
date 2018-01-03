/*
Copyright 2018 The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fake

import (
	stash "github.com/appscode/stash/apis/stash"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeRestics implements ResticInterface
type FakeRestics struct {
	Fake *FakeStash
	ns   string
}

var resticsResource = schema.GroupVersionResource{Group: "stash.appscode.com", Version: "", Resource: "restics"}

var resticsKind = schema.GroupVersionKind{Group: "stash.appscode.com", Version: "", Kind: "Restic"}

// Get takes name of the restic, and returns the corresponding restic object, and an error if there is any.
func (c *FakeRestics) Get(name string, options v1.GetOptions) (result *stash.Restic, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(resticsResource, c.ns, name), &stash.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*stash.Restic), err
}

// List takes label and field selectors, and returns the list of Restics that match those selectors.
func (c *FakeRestics) List(opts v1.ListOptions) (result *stash.ResticList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(resticsResource, resticsKind, c.ns, opts), &stash.ResticList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &stash.ResticList{}
	for _, item := range obj.(*stash.ResticList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested restics.
func (c *FakeRestics) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(resticsResource, c.ns, opts))

}

// Create takes the representation of a restic and creates it.  Returns the server's representation of the restic, and an error, if there is any.
func (c *FakeRestics) Create(restic *stash.Restic) (result *stash.Restic, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(resticsResource, c.ns, restic), &stash.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*stash.Restic), err
}

// Update takes the representation of a restic and updates it. Returns the server's representation of the restic, and an error, if there is any.
func (c *FakeRestics) Update(restic *stash.Restic) (result *stash.Restic, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(resticsResource, c.ns, restic), &stash.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*stash.Restic), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeRestics) UpdateStatus(restic *stash.Restic) (*stash.Restic, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(resticsResource, "status", c.ns, restic), &stash.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*stash.Restic), err
}

// Delete takes name of the restic and deletes it. Returns an error if one occurs.
func (c *FakeRestics) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(resticsResource, c.ns, name), &stash.Restic{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeRestics) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(resticsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &stash.ResticList{})
	return err
}

// Patch applies the patch and returns the patched restic.
func (c *FakeRestics) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *stash.Restic, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(resticsResource, c.ns, name, data, subresources...), &stash.Restic{})

	if obj == nil {
		return nil, err
	}
	return obj.(*stash.Restic), err
}
