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

package apiutil

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

type Cachable interface {
	GVK(gvk schema.GroupVersionKind) (bool, error)
	GVR(gvr schema.GroupVersionResource) (bool, error)
}

type cachable struct {
	resources map[string][]metav1.APIResource
}

var _ Cachable = &cachable{}

func NewCachable(cl discovery.DiscoveryInterface) (Cachable, error) {
	_, list, err := cl.ServerGroupsAndResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, err
	}
	m := map[string][]metav1.APIResource{}
	for _, rl := range list {
		m[rl.GroupVersion] = rl.APIResources
	}
	return &cachable{resources: m}, nil
}

func (c *cachable) GVK(gvk schema.GroupVersionKind) (bool, error) {
	rl, ok := c.resources[gvk.GroupVersion().String()]
	if !ok {
		return false, &meta.NoKindMatchError{
			GroupKind:        gvk.GroupKind(),
			SearchedVersions: []string{gvk.Version},
		}
	}
	for _, r := range rl {
		if r.Kind != gvk.Kind {
			continue
		}
		var canList, canWatch bool
		for _, verb := range r.Verbs {
			if verb == "list" {
				canList = true
			}
			if verb == "watch" {
				canWatch = true
			}
		}
		return canList && canWatch, nil
	}
	return false, &meta.NoKindMatchError{
		GroupKind:        gvk.GroupKind(),
		SearchedVersions: []string{gvk.Version},
	}
}

func (c *cachable) GVR(gvr schema.GroupVersionResource) (bool, error) {
	rl, ok := c.resources[gvr.GroupVersion().String()]
	if !ok {
		return false, &meta.NoResourceMatchError{
			PartialResource: gvr,
		}
	}
	for _, r := range rl {
		if r.Name != gvr.Resource {
			continue
		}
		var canList, canWatch bool
		for _, verb := range r.Verbs {
			if verb == "list" {
				canList = true
			}
			if verb == "watch" {
				canWatch = true
			}
		}
		return canList && canWatch, nil
	}
	return false, &meta.NoResourceMatchError{
		PartialResource: gvr,
	}
}
