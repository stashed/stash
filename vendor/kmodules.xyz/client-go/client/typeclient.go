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

package client

import (
	"context"
	"fmt"
	"strings"
	"sync"

	apiutil2 "kmodules.xyz/client-go/client/apiutil"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const listType = "List"

type typedClient struct {
	c        client.Client
	cachable apiutil2.Cachable
	*readerWrapper
}

var (
	_ client.Reader       = &typedClient{}
	_ client.Writer       = &typedClient{}
	_ client.StatusClient = &typedClient{}
	_ apiutil2.Cachable   = &typedClient{}
)

// GroupVersionKindFor returns the GroupVersionKind for the given object.
func (d *typedClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return d.c.GroupVersionKindFor(obj)
}

// IsObjectNamespaced returns true if the GroupVersionKind of the object is namespaced.
func (d *typedClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return d.c.IsObjectNamespaced(obj)
}

// Scheme returns the scheme this client is using.
func (d *typedClient) Scheme() *runtime.Scheme {
	return d.c.Scheme()
}

// RESTMapper returns the rest this client is using.
func (d *typedClient) RESTMapper() apimeta.RESTMapper {
	return d.c.RESTMapper()
}

type readerWrapper struct {
	c       client.Reader
	scheme  *runtime.Scheme
	typeMap map[schema.GroupVersionKind]schema.GroupVersionKind
	mu      sync.Mutex
}

var _ client.Reader = &readerWrapper{}

func (d *readerWrapper) getMappedType(gvk schema.GroupVersionKind) (schema.GroupVersionKind, bool) {
	d.mu.Lock()
	rawGVK, found := d.typeMap[gvk]
	d.mu.Unlock()
	return rawGVK, found
}

func (d *readerWrapper) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	gvk, err := apiutil.GVKForObject(obj, d.scheme)
	if err != nil {
		return err
	}

	rawGVK, found := d.getMappedType(gvk)
	if !found {
		return d.c.Get(ctx, key, obj, opts...)
	}

	ll, err := d.scheme.New(rawGVK)
	if err != nil {
		return err
	}
	llo := ll.(client.Object)
	err = d.c.Get(ctx, key, llo, opts...)
	if err != nil {
		return err
	}

	return d.scheme.Convert(llo, obj, nil)
}

func (d *readerWrapper) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	gvk, err := apiutil.GVKForObject(list, d.scheme)
	if err != nil {
		return err
	}
	if strings.HasSuffix(gvk.Kind, listType) && apimeta.IsListType(list) {
		gvk.Kind = gvk.Kind[:len(gvk.Kind)-4]
	}

	rawGVK, found := d.getMappedType(gvk)
	if !found {
		return d.c.List(ctx, list, opts...)
	}

	listGVK := rawGVK
	listGVK.Kind += listType

	ll, err := d.scheme.New(listGVK)
	if err != nil {
		return err
	}
	llo := ll.(client.ObjectList)
	err = d.c.List(ctx, llo, opts...)
	if err != nil {
		return err
	}

	list.SetResourceVersion(llo.GetResourceVersion())
	list.SetContinue(llo.GetContinue())
	list.SetSelfLink(llo.GetSelfLink())
	list.SetRemainingItemCount(llo.GetRemainingItemCount())

	items := make([]runtime.Object, 0, apimeta.LenList(llo))
	err = apimeta.EachListItem(llo, func(object runtime.Object) error {
		d2, err := d.scheme.New(gvk)
		if err != nil {
			return err
		}
		err = d.scheme.Convert(object, d2, nil)
		if err != nil {
			return err
		}
		items = append(items, d2)
		return nil
	})
	if err != nil {
		return err
	}
	return apimeta.SetList(list, items)
}

func (d *typedClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	gvk, err := apiutil.GVKForObject(obj, d.c.Scheme())
	if err != nil {
		return err
	}

	rawGVK, found := d.getMappedType(gvk)
	if !found {
		return d.c.Create(ctx, obj, opts...)
	}

	ll, err := d.c.Scheme().New(rawGVK)
	if err != nil {
		return err
	}
	llo := ll.(client.Object)
	err = d.Scheme().Convert(obj, llo, nil)
	if err != nil {
		return err
	}
	return d.c.Create(ctx, llo, opts...)
}

func (d *typedClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	gvk, err := apiutil.GVKForObject(obj, d.c.Scheme())
	if err != nil {
		return err
	}

	rawGVK, found := d.getMappedType(gvk)
	if !found {
		return d.c.Delete(ctx, obj, opts...)
	}

	ll, err := d.c.Scheme().New(rawGVK)
	if err != nil {
		return err
	}
	llo := ll.(client.Object)
	llo.SetNamespace(obj.GetNamespace())
	llo.SetName(obj.GetName())
	llo.SetLabels(obj.GetLabels())
	return d.c.Delete(ctx, llo, opts...)
}

func (d *typedClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	gvk, err := apiutil.GVKForObject(obj, d.c.Scheme())
	if err != nil {
		return err
	}

	rawGVK, found := d.getMappedType(gvk)
	if !found {
		return d.c.Update(ctx, obj, opts...)
	}

	ll, err := d.c.Scheme().New(rawGVK)
	if err != nil {
		return err
	}
	llo := ll.(client.Object)
	err = d.Scheme().Convert(obj, llo, nil)
	if err != nil {
		return err
	}
	return d.c.Update(ctx, llo, opts...)
}

func (d *typedClient) Apply(ctx context.Context, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
	panic("not implemented")
}

func (d *typedClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	gvk, err := apiutil.GVKForObject(obj, d.c.Scheme())
	if err != nil {
		return err
	}

	rawGVK, found := d.getMappedType(gvk)
	if !found {
		return d.c.Patch(ctx, obj, patch, opts...)
	}

	ll, err := d.c.Scheme().New(rawGVK)
	if err != nil {
		return err
	}
	llo := ll.(client.Object)
	llo.SetNamespace(obj.GetNamespace())
	llo.SetName(obj.GetName())
	llo.SetLabels(obj.GetLabels())
	return d.c.Patch(ctx, llo, patch, opts...)
}

func (d *typedClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	gvk, err := apiutil.GVKForObject(obj, d.c.Scheme())
	if err != nil {
		return err
	}

	rawGVK, found := d.getMappedType(gvk)
	if !found {
		return d.c.DeleteAllOf(ctx, obj, opts...)
	}

	ll, err := d.c.Scheme().New(rawGVK)
	if err != nil {
		return err
	}
	llo := ll.(client.Object)
	llo.SetNamespace(obj.GetNamespace())
	llo.SetName(obj.GetName())
	llo.SetLabels(obj.GetLabels())
	return d.c.DeleteAllOf(ctx, llo, opts...)
}

func (d *typedClient) Status() client.StatusWriter {
	return &typedStatusWriter{client: d}
}

// typedStatusWriter is client.StatusWriter that writes status subresource.
type typedStatusWriter struct {
	client *typedClient
}

// ensure typedStatusWriter implements client.StatusWriter.
var _ client.StatusWriter = &typedStatusWriter{}

func (sw *typedStatusWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	gvk, err := apiutil.GVKForObject(obj, sw.client.c.Scheme())
	if err != nil {
		return err
	}

	rawGVK, found := sw.client.getMappedType(gvk)
	if !found {
		return sw.client.c.Status().Create(ctx, obj, subResource, opts...)
	}

	ll, err := sw.client.Scheme().New(rawGVK)
	if err != nil {
		return err
	}
	llo := ll.(client.Object)
	err = sw.client.Scheme().Convert(obj, llo, nil)
	if err != nil {
		return err
	}
	return sw.client.c.Status().Create(ctx, llo, subResource, opts...)
}

func (sw *typedStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	gvk, err := apiutil.GVKForObject(obj, sw.client.c.Scheme())
	if err != nil {
		return err
	}

	rawGVK, found := sw.client.getMappedType(gvk)
	if !found {
		return sw.client.c.Status().Update(ctx, obj, opts...)
	}

	ll, err := sw.client.Scheme().New(rawGVK)
	if err != nil {
		return err
	}
	llo := ll.(client.Object)
	err = sw.client.Scheme().Convert(obj, llo, nil)
	if err != nil {
		return err
	}
	return sw.client.c.Status().Update(ctx, llo, opts...)
}

func (sw *typedStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	gvk, err := apiutil.GVKForObject(obj, sw.client.c.Scheme())
	if err != nil {
		return err
	}

	rawGVK, found := sw.client.getMappedType(gvk)
	if !found {
		return sw.client.c.Status().Patch(ctx, obj, patch, opts...)
	}

	ll, err := sw.client.c.Scheme().New(rawGVK)
	if err != nil {
		return err
	}
	llo := ll.(client.Object)
	llo.SetNamespace(obj.GetNamespace())
	llo.SetName(obj.GetName())
	llo.SetLabels(obj.GetLabels())
	return sw.client.c.Status().Patch(ctx, llo, patch, opts...)
}

func (d *typedClient) SubResource(subResource string) client.SubResourceClient {
	return d.c.SubResource(subResource)
}

func (d *typedClient) GVK(gvk schema.GroupVersionKind) (bool, error) {
	d.mu.Lock()
	rawGVK, found := d.typeMap[gvk]
	d.mu.Unlock()

	if !found {
		return d.cachable.GVK(gvk)
	}
	return d.cachable.GVK(rawGVK)
}

func (d *typedClient) GVR(gvr schema.GroupVersionResource) (bool, error) {
	gvk, err := d.c.RESTMapper().KindFor(schema.GroupVersionResource{
		Group:    gvr.Group,
		Version:  "",
		Resource: gvr.Resource,
	})
	if err != nil {
		return false, err
	}
	return d.GVK(gvk)
}

func BuildTypeMap(kc client.Client, gvks ...schema.GroupVersionKind) (map[schema.GroupVersionKind]schema.GroupVersionKind, error) {
	tm := map[schema.GroupVersionKind]schema.GroupVersionKind{}

	for _, gvk := range gvks {
		mappings, err := kc.RESTMapper().RESTMappings(gvk.GroupKind())
		if err != nil {
			return nil, err
		}

		var found bool
		for _, mapping := range mappings {
			if mapping.GroupVersionKind == gvk {
				found = true
				break
			}
		}
		if !found {
			for _, mapping := range mappings {

				in, err := kc.Scheme().New(gvk)
				if err != nil {
					return nil, err
				}
				out, err := kc.Scheme().New(mapping.GroupVersionKind)
				if err != nil {
					return nil, err
				}
				if err := kc.Scheme().Convert(in, out, nil); err == nil {
					tm[gvk] = mapping.GroupVersionKind
					found = true
					break
				}
			}
		}
		if !found {
			return nil, fmt.Errorf("type mapping not found for %+v", gvk)
		}
	}

	return tm, nil
}

func NewAutoConvertClient(gvks ...schema.GroupVersionKind) client.NewClientFunc {
	return func(config *restclient.Config, options client.Options) (client.Client, error) {
		c, err := client.New(config, options)
		if err != nil {
			return nil, err
		}
		cachable, err := apiutil2.NewDynamicCachable(config)
		if err != nil {
			return nil, err
		}
		tm, err := BuildTypeMap(c, gvks...)
		if err != nil {
			return nil, err
		}
		tc := &typedClient{
			c:        c,
			cachable: cachable,
			readerWrapper: &readerWrapper{
				c:       c,
				scheme:  c.Scheme(),
				typeMap: tm,
			},
		}

		co := NewDelegatingClientInput{
			Client:   tc,
			Cachable: tc,
		}
		if options.Cache != nil {
			co.CacheReader = &readerWrapper{
				c:       options.Cache.Reader,
				scheme:  c.Scheme(),
				typeMap: tm,
			}
			co.UncachedObjects = options.Cache.DisableFor
			co.CacheUnstructured = options.Cache.Unstructured // cache unstructured objects
		}
		return NewDelegatingClient(co)
	}
}
