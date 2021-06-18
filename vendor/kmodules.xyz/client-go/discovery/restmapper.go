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

package discovery

import (
	"fmt"
	"strings"
	"sync"

	kmapi "kmodules.xyz/client-go/api/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func APIResourceForGVK(client discovery.DiscoveryInterface, gvk schema.GroupVersionKind) (metav1.APIResource, error) {
	resourceList, err := client.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if discovery.IsGroupDiscoveryFailedError(err) {
		klog.Errorf("Skipping failed API Groups: %v", err)
	} else if err != nil {
		return metav1.APIResource{}, err
	}
	var resources []metav1.APIResource
	for _, resource := range resourceList.APIResources {
		if resource.Kind == gvk.Kind { // match kind
			resource.Group = gvk.Group
			resource.Version = gvk.Version
			resources = append(resources, resource)
		}
	}
	resources = FilterAPISubResources(resources) // ignore sub-resources
	switch len(resources) {
	case 0:
		return metav1.APIResource{}, fmt.Errorf("APIResource not found for gvk=%v", gvk)
	case 1:
		return resources[0], nil
	default:
		var matches []schema.GroupVersionResource
		for _, resource := range resources {
			matches = append(matches, schema.GroupVersionResource{
				Group:    resource.Group,
				Version:  resource.Version,
				Resource: resource.Name,
			})
		}
		return metav1.APIResource{}, &meta.AmbiguousResourceError{PartialResource: matches[0], MatchingResources: matches}
	}
}

func ResourceForGVK(client discovery.DiscoveryInterface, gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	return NewResourceMapper(NewRestMapper(client)).GVR(gvk)
}

func FilterAPISubResources(resources []metav1.APIResource) []metav1.APIResource {
	var filtered []metav1.APIResource
	for _, res := range resources {
		if !strings.ContainsRune(res.Name, '/') {
			filtered = append(filtered, res)
		}
	}
	return filtered
}

func FilterSubResources(gvrs []schema.GroupVersionResource) []schema.GroupVersionResource {
	var filtered []schema.GroupVersionResource
	for _, res := range gvrs {
		if !strings.ContainsRune(res.Resource, '/') {
			filtered = append(filtered, res)
		}
	}
	return filtered
}

func NewRestMapper(client discovery.DiscoveryInterface) meta.RESTMapper {
	return restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(client))
}

type ResourceMapper interface {
	ResourceIDForGVK(gvk schema.GroupVersionKind) (*kmapi.ResourceID, error)
	ResourceIDForGVR(gvr schema.GroupVersionResource) (*kmapi.ResourceID, error)
	GVR(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error)
	TypeMeta(gvr schema.GroupVersionResource) (metav1.TypeMeta, error)
	GVK(gvr schema.GroupVersionResource) (schema.GroupVersionKind, error)
	IsNamespaced(gvr schema.GroupVersionResource) (bool, error)
	IsPreferred(gvr schema.GroupVersionResource) (bool, error)
	Preferred(gvr schema.GroupVersionResource) (schema.GroupVersionResource, error)
	ExistsGVR(gvr schema.GroupVersionResource) (bool, error)
	ExistsGVK(gvk schema.GroupVersionKind) (bool, error)
	Reset()
}

type resourcemapper struct {
	mapper meta.RESTMapper

	// Keeping this cache unbounded, since the # of unique GVKs will be at most few thousands
	cache map[schema.GroupVersionKind]*kmapi.ResourceID
	lock  sync.RWMutex
}

var _ ResourceMapper = &resourcemapper{}

func NewResourceMapper(mapper meta.RESTMapper) ResourceMapper {
	return &resourcemapper{mapper: mapper, cache: map[schema.GroupVersionKind]*kmapi.ResourceID{}}
}

func NewDynamicResourceMapper(cfg *rest.Config, opts ...apiutil.DynamicRESTMapperOption) (ResourceMapper, error) {
	mapper, err := apiutil.NewDynamicRESTMapper(cfg, opts...)
	if err != nil {
		return nil, err
	}
	return &resourcemapper{mapper: mapper, cache: map[schema.GroupVersionKind]*kmapi.ResourceID{}}, nil
}

func (m *resourcemapper) ResourceIDForGVK(gvk schema.GroupVersionKind) (*kmapi.ResourceID, error) {
	m.lock.RLock()
	rid, ok := m.cache[gvk]
	m.lock.RUnlock()
	if ok {
		return rid, nil
	}

	mapping, err := m.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}
	scope := kmapi.ClusterScoped
	if mapping.Scope == meta.RESTScopeNamespace {
		scope = kmapi.NamespaceScoped
	}
	rid = &kmapi.ResourceID{
		Group:   gvk.Group,
		Version: gvk.Version,
		Name:    mapping.Resource.Resource,
		Kind:    gvk.Kind,
		Scope:   scope,
	}
	m.lock.Lock()
	m.cache[gvk] = rid
	m.lock.Unlock()
	return rid, nil
}

func (m *resourcemapper) ResourceIDForGVR(gvr schema.GroupVersionResource) (*kmapi.ResourceID, error) {
	gvk, err := m.mapper.KindFor(gvr)
	if err != nil {
		return nil, err
	}

	m.lock.RLock()
	rid, ok := m.cache[gvk]
	m.lock.RUnlock()
	if ok {
		return rid, nil
	}

	mapping, err := m.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}
	scope := kmapi.ClusterScoped
	if mapping.Scope == meta.RESTScopeNamespace {
		scope = kmapi.NamespaceScoped
	}
	rid = &kmapi.ResourceID{
		Group:   gvr.Group,
		Version: gvr.Version,
		Name:    gvr.Resource,
		Kind:    gvk.Kind,
		Scope:   scope,
	}
	m.lock.Lock()
	m.cache[gvk] = rid
	m.lock.Unlock()
	return rid, nil
}

func (m *resourcemapper) GVR(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	mapping, err := m.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	return mapping.Resource, nil
}

func (m *resourcemapper) TypeMeta(gvr schema.GroupVersionResource) (metav1.TypeMeta, error) {
	gvk, err := m.mapper.KindFor(gvr)
	if err != nil {
		return metav1.TypeMeta{}, err
	}
	return metav1.TypeMeta{
		APIVersion: gvr.GroupVersion().String(),
		Kind:       gvk.Kind,
	}, nil
}

func (m *resourcemapper) GVK(gvr schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	gvk, err := m.mapper.KindFor(gvr)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	return gvk, nil
}

func (m *resourcemapper) IsNamespaced(gvr schema.GroupVersionResource) (bool, error) {
	gvk, err := m.mapper.KindFor(gvr)
	if err != nil {
		return false, err
	}
	mapping, err := m.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return false, err
	}
	return mapping.Scope == meta.RESTScopeNamespace, nil
}

func (m *resourcemapper) IsPreferred(gvr schema.GroupVersionResource) (bool, error) {
	gvrs, err := m.mapper.ResourcesFor(gvr.GroupResource().WithVersion(""))
	if err != nil {
		return false, err
	}
	return gvrs[0] == gvr, nil
}

func (m *resourcemapper) Preferred(gvr schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	gvrs, err := m.mapper.ResourcesFor(gvr.GroupResource().WithVersion(""))
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	return gvrs[0], nil
}

func (m *resourcemapper) ExistsGVR(gvr schema.GroupVersionResource) (bool, error) {
	_, err := m.mapper.ResourceFor(gvr)
	if meta.IsNoMatchError(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (m *resourcemapper) ExistsGVK(gvk schema.GroupVersionKind) (bool, error) {
	_, err := m.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if meta.IsNoMatchError(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (m *resourcemapper) Reset() {
	type ResetCache interface {
		Reset()
	}
	if c, ok := m.mapper.(ResetCache); ok {
		c.Reset()
	}
}
