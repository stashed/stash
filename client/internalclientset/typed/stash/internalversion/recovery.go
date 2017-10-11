/*
Copyright 2017 The Stash Authors.

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

package internalversion

import (
	stash "github.com/appscode/stash/apis/stash"
	scheme "github.com/appscode/stash/client/internalclientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// RecoveriesGetter has a method to return a RecoveryInterface.
// A group's client should implement this interface.
type RecoveriesGetter interface {
	Recoveries(namespace string) RecoveryInterface
}

// RecoveryInterface has methods to work with Recovery resources.
type RecoveryInterface interface {
	Create(*stash.Recovery) (*stash.Recovery, error)
	Update(*stash.Recovery) (*stash.Recovery, error)
	UpdateStatus(*stash.Recovery) (*stash.Recovery, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*stash.Recovery, error)
	List(opts v1.ListOptions) (*stash.RecoveryList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *stash.Recovery, err error)
	RecoveryExpansion
}

// recoveries implements RecoveryInterface
type recoveries struct {
	client rest.Interface
	ns     string
}

// newRecoveries returns a Recoveries
func newRecoveries(c *StashClient, namespace string) *recoveries {
	return &recoveries{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Create takes the representation of a recovery and creates it.  Returns the server's representation of the recovery, and an error, if there is any.
func (c *recoveries) Create(recovery *stash.Recovery) (result *stash.Recovery, err error) {
	result = &stash.Recovery{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("recoveries").
		Body(recovery).
		Do().
		Into(result)
	return
}

// Update takes the representation of a recovery and updates it. Returns the server's representation of the recovery, and an error, if there is any.
func (c *recoveries) Update(recovery *stash.Recovery) (result *stash.Recovery, err error) {
	result = &stash.Recovery{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("recoveries").
		Name(recovery.Name).
		Body(recovery).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclientstatus=false comment above the type to avoid generating UpdateStatus().

func (c *recoveries) UpdateStatus(recovery *stash.Recovery) (result *stash.Recovery, err error) {
	result = &stash.Recovery{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("recoveries").
		Name(recovery.Name).
		SubResource("status").
		Body(recovery).
		Do().
		Into(result)
	return
}

// Delete takes name of the recovery and deletes it. Returns an error if one occurs.
func (c *recoveries) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("recoveries").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *recoveries) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("recoveries").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the recovery, and returns the corresponding recovery object, and an error if there is any.
func (c *recoveries) Get(name string, options v1.GetOptions) (result *stash.Recovery, err error) {
	result = &stash.Recovery{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("recoveries").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Recoveries that match those selectors.
func (c *recoveries) List(opts v1.ListOptions) (result *stash.RecoveryList, err error) {
	result = &stash.RecoveryList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("recoveries").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested recoveries.
func (c *recoveries) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("recoveries").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched recovery.
func (c *recoveries) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *stash.Recovery, err error) {
	result = &stash.Recovery{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("recoveries").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
