/*
Copyright AppsCode Inc. and Contributors

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

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"context"
	json "encoding/json"
	"fmt"
	"time"

	v1 "kmodules.xyz/openshift/apis/security/v1"
	securityv1 "kmodules.xyz/openshift/client/applyconfiguration/security/v1"
	scheme "kmodules.xyz/openshift/client/clientset/versioned/scheme"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// SecurityContextConstraintsGetter has a method to return a SecurityContextConstraintsInterface.
// A group's client should implement this interface.
type SecurityContextConstraintsGetter interface {
	SecurityContextConstraints() SecurityContextConstraintsInterface
}

// SecurityContextConstraintsInterface has methods to work with SecurityContextConstraints resources.
type SecurityContextConstraintsInterface interface {
	Create(ctx context.Context, securityContextConstraints *v1.SecurityContextConstraints, opts metav1.CreateOptions) (*v1.SecurityContextConstraints, error)
	Update(ctx context.Context, securityContextConstraints *v1.SecurityContextConstraints, opts metav1.UpdateOptions) (*v1.SecurityContextConstraints, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.SecurityContextConstraints, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.SecurityContextConstraintsList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.SecurityContextConstraints, err error)
	Apply(ctx context.Context, securityContextConstraints *securityv1.SecurityContextConstraintsApplyConfiguration, opts metav1.ApplyOptions) (result *v1.SecurityContextConstraints, err error)
	SecurityContextConstraintsExpansion
}

// securityContextConstraints implements SecurityContextConstraintsInterface
type securityContextConstraints struct {
	client rest.Interface
}

// newSecurityContextConstraints returns a SecurityContextConstraints
func newSecurityContextConstraints(c *SecurityV1Client) *securityContextConstraints {
	return &securityContextConstraints{
		client: c.RESTClient(),
	}
}

// Get takes name of the securityContextConstraints, and returns the corresponding securityContextConstraints object, and an error if there is any.
func (c *securityContextConstraints) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.SecurityContextConstraints, err error) {
	result = &v1.SecurityContextConstraints{}
	err = c.client.Get().
		Resource("securitycontextconstraints").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of SecurityContextConstraints that match those selectors.
func (c *securityContextConstraints) List(ctx context.Context, opts metav1.ListOptions) (result *v1.SecurityContextConstraintsList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.SecurityContextConstraintsList{}
	err = c.client.Get().
		Resource("securitycontextconstraints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested securityContextConstraints.
func (c *securityContextConstraints) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("securitycontextconstraints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a securityContextConstraints and creates it.  Returns the server's representation of the securityContextConstraints, and an error, if there is any.
func (c *securityContextConstraints) Create(ctx context.Context, securityContextConstraints *v1.SecurityContextConstraints, opts metav1.CreateOptions) (result *v1.SecurityContextConstraints, err error) {
	result = &v1.SecurityContextConstraints{}
	err = c.client.Post().
		Resource("securitycontextconstraints").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(securityContextConstraints).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a securityContextConstraints and updates it. Returns the server's representation of the securityContextConstraints, and an error, if there is any.
func (c *securityContextConstraints) Update(ctx context.Context, securityContextConstraints *v1.SecurityContextConstraints, opts metav1.UpdateOptions) (result *v1.SecurityContextConstraints, err error) {
	result = &v1.SecurityContextConstraints{}
	err = c.client.Put().
		Resource("securitycontextconstraints").
		Name(securityContextConstraints.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(securityContextConstraints).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the securityContextConstraints and deletes it. Returns an error if one occurs.
func (c *securityContextConstraints) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Resource("securitycontextconstraints").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *securityContextConstraints) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("securitycontextconstraints").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched securityContextConstraints.
func (c *securityContextConstraints) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.SecurityContextConstraints, err error) {
	result = &v1.SecurityContextConstraints{}
	err = c.client.Patch(pt).
		Resource("securitycontextconstraints").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied securityContextConstraints.
func (c *securityContextConstraints) Apply(ctx context.Context, securityContextConstraints *securityv1.SecurityContextConstraintsApplyConfiguration, opts metav1.ApplyOptions) (result *v1.SecurityContextConstraints, err error) {
	if securityContextConstraints == nil {
		return nil, fmt.Errorf("securityContextConstraints provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(securityContextConstraints)
	if err != nil {
		return nil, err
	}
	name := securityContextConstraints.Name
	if name == nil {
		return nil, fmt.Errorf("securityContextConstraints.Name must be provided to Apply")
	}
	result = &v1.SecurityContextConstraints{}
	err = c.client.Patch(types.ApplyPatchType).
		Resource("securitycontextconstraints").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
