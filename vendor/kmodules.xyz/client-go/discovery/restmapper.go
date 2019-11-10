/*
Copyright The Kmodules Authors.

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
	"reflect"
	"sort"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

func DetectResource(restmapper *DefaultRESTMapper, obj interface{}) (schema.GroupVersionResource, error) {
	gvk, err := guessGVK(obj)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	resources, err := restmapper.ResourcesForKind(gvk)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	result := make([]schema.GroupVersionResource, 0, len(resources))
	for _, resource := range resources {
		if strings.ContainsRune(resource.Resource, '/') {
			continue
		}
		result = append(result, resource)
	}
	if len(result) == 1 {
		return result[0], nil
	}
	return schema.GroupVersionResource{}, &AmbiguousResourceError{PartialResource: gvk, MatchingResources: resources}
}

func APIResourceForGVK(client discovery.DiscoveryInterface, input schema.GroupVersionKind) (metav1.APIResource, error) {
	resourceList, err := client.ServerResourcesForGroupVersion(input.GroupVersion().String())
	if discovery.IsGroupDiscoveryFailedError(err) {
		glog.Errorf("Skipping failed API Groups: %v", err)
	} else if err != nil {
		return metav1.APIResource{}, err
	}
	var resources []metav1.APIResource
	for _, resource := range resourceList.APIResources {
		if resource.Kind == input.Kind { // match kind
			resource.Group = input.Group
			resource.Version = input.Version
			resources = append(resources, resource)
		}
	}
	resources = FilterAPISubResources(resources) // ignore sub-resources
	if len(resources) == 1 {
		return resources[0], nil
	}

	var matches []schema.GroupVersionResource
	for _, resource := range resources {
		matches = append(matches, schema.GroupVersionResource{
			Group:    resource.Group,
			Version:  resource.Version,
			Resource: resource.Name,
		})
	}
	return metav1.APIResource{}, &AmbiguousResourceError{PartialResource: input, MatchingResources: matches}
}

func ResourceForGVK(client discovery.DiscoveryInterface, input schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	resourceList, err := client.ServerResourcesForGroupVersion(input.GroupVersion().String())
	if discovery.IsGroupDiscoveryFailedError(err) {
		glog.Errorf("Skipping failed API Groups: %v", err)
	} else if err != nil {
		return schema.GroupVersionResource{}, err
	}
	var resources []schema.GroupVersionResource
	for _, resource := range resourceList.APIResources {
		if resource.Kind == input.Kind { // match kind
			resources = append(resources, input.GroupVersion().WithResource(resource.Name))
		}
	}
	resources = FilterSubResources(resources) // ignore sub-resources
	if len(resources) == 1 {
		return resources[0], nil
	}
	return schema.GroupVersionResource{}, &AmbiguousResourceError{PartialResource: input, MatchingResources: resources}
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

func FilterSubResources(resources []schema.GroupVersionResource) []schema.GroupVersionResource {
	var filtered []schema.GroupVersionResource
	for _, res := range resources {
		if !strings.ContainsRune(res.Resource, '/') {
			filtered = append(filtered, res)
		}
	}
	return filtered
}

func LoadRestMapper(client discovery.DiscoveryInterface) (*DefaultRESTMapper, error) {
	restMapper := NewDefaultRESTMapper([]schema.GroupVersion{})

	_, resourceLists, err := client.ServerGroupsAndResources()
	if discovery.IsGroupDiscoveryFailedError(err) {
		glog.Errorf("Skipping failed API Groups: %v", err)
	} else if err != nil {
		return nil, err
	}
	for _, resourceList := range resourceLists {
		for _, resource := range resourceList.APIResources {
			gv, _ := schema.ParseGroupVersion(resourceList.GroupVersion)
			plural := gv.WithResource(resource.Name)
			singular := gv.WithResource(resource.SingularName)
			gvk := gv.WithKind(resource.Kind)
			restMapper.AddSpecific(gvk, plural, singular)
		}
	}
	return restMapper, nil
}

func guessGVK(obj interface{}) (schema.GroupVersionKind, error) {
	if m, err := meta.TypeAccessor(obj); err == nil {
		return schema.FromAPIVersionAndKind(m.GetAPIVersion(), m.GetKind()), nil
	}

	val, err := conversion.EnforcePtr(obj)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}

	pp := pkgPath(val)
	parts := strings.Split(pp, "/")
	if len(parts) < 2 {
		return schema.GroupVersionKind{}, errors.Errorf("failed to guess GroupVersion from package path %s", pp)
	}
	group := parts[len(parts)-2]
	if strings.HasPrefix(pp, "k8s.io/api") && group == "core" {
		group = ""
	}
	version := parts[len(parts)-1]

	return schema.GroupVersionKind{Group: group, Version: version, Kind: val.Type().Name()}, nil
}

func pkgPath(val reflect.Value) string {
	p := val.Type().PkgPath()
	idx := strings.LastIndex(p, "/vendor/")
	if idx > -1 {
		p = p[idx+len("/vendor/"):]
	}
	return p
}

// ref: https://github.com/kubernetes/apimachinery/blob/bfe7139b14565386b4470736ca765bf21616b60e/pkg/api/meta/restmapper.go#L72
type DefaultRESTMapper struct {
	defaultGroupVersions []schema.GroupVersion

	resourceToKind       map[schema.GroupVersionResource]schema.GroupVersionKind
	kindToPluralResource map[schema.GroupVersionKind]schema.GroupVersionResource
	singularToPlural     map[schema.GroupVersionResource]schema.GroupVersionResource
	pluralToSingular     map[schema.GroupVersionResource]schema.GroupVersionResource
}

func NewDefaultRESTMapper(defaultGroupVersions []schema.GroupVersion) *DefaultRESTMapper {
	resourceToKind := make(map[schema.GroupVersionResource]schema.GroupVersionKind)
	kindToPluralResource := make(map[schema.GroupVersionKind]schema.GroupVersionResource)
	singularToPlural := make(map[schema.GroupVersionResource]schema.GroupVersionResource)
	pluralToSingular := make(map[schema.GroupVersionResource]schema.GroupVersionResource)
	// TODO: verify name mappings work correctly when versions differ

	return &DefaultRESTMapper{
		resourceToKind:       resourceToKind,
		kindToPluralResource: kindToPluralResource,
		defaultGroupVersions: defaultGroupVersions,
		singularToPlural:     singularToPlural,
		pluralToSingular:     pluralToSingular,
	}
}

func (m *DefaultRESTMapper) AddSpecific(kind schema.GroupVersionKind, plural, singular schema.GroupVersionResource) {
	m.singularToPlural[singular] = plural
	m.pluralToSingular[plural] = singular

	m.resourceToKind[singular] = kind
	m.resourceToKind[plural] = kind

	m.kindToPluralResource[kind] = plural
}

func (m *DefaultRESTMapper) ResourcesForKind(input schema.GroupVersionKind) ([]schema.GroupVersionResource, error) {
	gvk := coerceKindForMatching(input)

	hasResource := len(gvk.Kind) > 0
	hasGroup := len(gvk.Group) > 0
	hasVersion := len(gvk.Version) > 0

	if !hasResource {
		return nil, errors.Errorf("a resource must be present, got: %v", gvk)
	}

	var ret []schema.GroupVersionResource
	switch {
	case hasGroup:
		// given a group, prefer an exact match.  If you don't find one, resort to a prefix match on group
		foundExactMatch := false
		requestedGroupKind := gvk.GroupKind()
		for plural := range m.pluralToSingular {
			kind, ok := m.resourceToKind[plural]
			if !ok {
				continue
			}
			if kind.GroupKind() == requestedGroupKind && (!hasVersion || kind.Version == gvk.Version) {
				foundExactMatch = true
				ret = append(ret, plural)
			}
		}

		// if you didn't find an exact match, match on group prefixing. This allows storageclass.storage to match
		// storageclass.storage.k8s.io
		if !foundExactMatch {
			for plural := range m.pluralToSingular {
				if !strings.HasPrefix(plural.Group, requestedGroupKind.Group) {
					continue
				}
				kind, ok := m.resourceToKind[plural]
				if !ok {
					continue
				}
				if kind.Kind == requestedGroupKind.Kind && (!hasVersion || kind.Version == gvk.Version) {
					ret = append(ret, plural)
				}
			}
		}

	case hasVersion:
		for plural := range m.pluralToSingular {
			kind, ok := m.resourceToKind[plural]
			if !ok {
				continue
			}
			if kind.Version == gvk.Version && kind.Kind == gvk.Kind {
				ret = append(ret, plural)
			}
		}

	default:
		for plural := range m.pluralToSingular {
			kind, ok := m.resourceToKind[plural]
			if !ok {
				continue
			}
			if kind.Kind == gvk.Kind {
				ret = append(ret, plural)
			}
		}
	}

	if len(ret) == 0 {
		return nil, errors.Errorf("no matches for %v", gvk)
	}

	sort.Sort(resourceByPreferredGroupVersion{ret, m.defaultGroupVersions})
	return ret, nil
}

func (m *DefaultRESTMapper) ResourceForKind(input schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	resources, err := m.ResourcesForKind(input)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	if len(resources) == 1 {
		return resources[0], nil
	}

	return schema.GroupVersionResource{}, &AmbiguousResourceError{PartialResource: input, MatchingResources: resources}
}

// coerceKindForMatching makes the resource lower case and converts internal versions to unspecified (legacy behavior)
func coerceKindForMatching(gvk schema.GroupVersionKind) schema.GroupVersionKind {
	if gvk.Version == runtime.APIVersionInternal {
		gvk.Version = ""
	}
	return gvk
}

// coerceResourceForMatching makes the resource lower case and converts internal versions to unspecified (legacy behavior)
func coerceResourceForMatching(resource schema.GroupVersionResource) schema.GroupVersionResource {
	resource.Resource = strings.ToLower(resource.Resource)
	if resource.Version == runtime.APIVersionInternal {
		resource.Version = ""
	}

	return resource
}

func (m *DefaultRESTMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	resource := coerceResourceForMatching(input)

	hasResource := len(resource.Resource) > 0
	hasGroup := len(resource.Group) > 0
	hasVersion := len(resource.Version) > 0

	if !hasResource {
		return nil, errors.Errorf("a resource must be present, got: %v", resource)
	}

	var ret []schema.GroupVersionResource
	switch {
	case hasGroup && hasVersion:
		// fully qualified.  Find the exact match
		for plural, singular := range m.pluralToSingular {
			if singular == resource {
				ret = append(ret, plural)
				break
			}
			if plural == resource {
				ret = append(ret, plural)
				break
			}
		}

	case hasGroup:
		// given a group, prefer an exact match.  If you don't find one, resort to a prefix match on group
		foundExactMatch := false
		requestedGroupResource := resource.GroupResource()
		for plural, singular := range m.pluralToSingular {
			if singular.GroupResource() == requestedGroupResource {
				foundExactMatch = true
				ret = append(ret, plural)
			}
			if plural.GroupResource() == requestedGroupResource {
				foundExactMatch = true
				ret = append(ret, plural)
			}
		}

		// if you didn't find an exact match, match on group prefixing. This allows storageclass.storage to match
		// storageclass.storage.k8s.io
		if !foundExactMatch {
			for plural, singular := range m.pluralToSingular {
				if !strings.HasPrefix(plural.Group, requestedGroupResource.Group) {
					continue
				}
				if singular.Resource == requestedGroupResource.Resource {
					ret = append(ret, plural)
				}
				if plural.Resource == requestedGroupResource.Resource {
					ret = append(ret, plural)
				}
			}

		}

	case hasVersion:
		for plural, singular := range m.pluralToSingular {
			if singular.Version == resource.Version && singular.Resource == resource.Resource {
				ret = append(ret, plural)
			}
			if plural.Version == resource.Version && plural.Resource == resource.Resource {
				ret = append(ret, plural)
			}
		}

	default:
		for plural, singular := range m.pluralToSingular {
			if singular.Resource == resource.Resource {
				ret = append(ret, plural)
			}
			if plural.Resource == resource.Resource {
				ret = append(ret, plural)
			}
		}
	}

	if len(ret) == 0 {
		return nil, errors.Errorf("no matches for %v", resource)
	}

	sort.Sort(resourceByPreferredGroupVersion{ret, m.defaultGroupVersions})
	return ret, nil
}

func (m *DefaultRESTMapper) ResourceFor(resource schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	resources, err := m.ResourcesFor(resource)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	if len(resources) == 1 {
		return resources[0], nil
	}

	return schema.GroupVersionResource{}, &meta.AmbiguousResourceError{PartialResource: resource, MatchingResources: resources}
}

func (m *DefaultRESTMapper) KindsFor(input schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	resource := coerceResourceForMatching(input)

	hasResource := len(resource.Resource) > 0
	hasGroup := len(resource.Group) > 0
	hasVersion := len(resource.Version) > 0

	if !hasResource {
		return nil, errors.Errorf("a resource must be present, got: %v", resource)
	}

	var ret []schema.GroupVersionKind
	switch {
	// fully qualified.  Find the exact match
	case hasGroup && hasVersion:
		kind, exists := m.resourceToKind[resource]
		if exists {
			ret = append(ret, kind)
		}

	case hasGroup:
		foundExactMatch := false
		requestedGroupResource := resource.GroupResource()
		for currResource, currKind := range m.resourceToKind {
			if currResource.GroupResource() == requestedGroupResource {
				foundExactMatch = true
				ret = append(ret, currKind)
			}
		}

		// if you didn't find an exact match, match on group prefixing. This allows storageclass.storage to match
		// storageclass.storage.k8s.io
		if !foundExactMatch {
			for currResource, currKind := range m.resourceToKind {
				if !strings.HasPrefix(currResource.Group, requestedGroupResource.Group) {
					continue
				}
				if currResource.Resource == requestedGroupResource.Resource {
					ret = append(ret, currKind)
				}
			}

		}

	case hasVersion:
		for currResource, currKind := range m.resourceToKind {
			if currResource.Version == resource.Version && currResource.Resource == resource.Resource {
				ret = append(ret, currKind)
			}
		}

	default:
		for currResource, currKind := range m.resourceToKind {
			if currResource.Resource == resource.Resource {
				ret = append(ret, currKind)
			}
		}
	}

	if len(ret) == 0 {
		return nil, errors.Errorf("no matches for %v", input)
	}

	sort.Sort(kindByPreferredGroupVersion{ret, m.defaultGroupVersions})
	return ret, nil
}

func (m *DefaultRESTMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	kinds, err := m.KindsFor(resource)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	if len(kinds) == 1 {
		return kinds[0], nil
	}

	return schema.GroupVersionKind{}, &meta.AmbiguousResourceError{PartialResource: resource, MatchingKinds: kinds}
}

type kindByPreferredGroupVersion struct {
	list      []schema.GroupVersionKind
	sortOrder []schema.GroupVersion
}

func (o kindByPreferredGroupVersion) Len() int      { return len(o.list) }
func (o kindByPreferredGroupVersion) Swap(i, j int) { o.list[i], o.list[j] = o.list[j], o.list[i] }
func (o kindByPreferredGroupVersion) Less(i, j int) bool {
	lhs := o.list[i]
	rhs := o.list[j]
	if lhs == rhs {
		return false
	}

	if lhs.GroupVersion() == rhs.GroupVersion() {
		return lhs.Kind < rhs.Kind
	}

	// otherwise, the difference is in the GroupVersion, so we need to sort with respect to the preferred order
	lhsIndex := -1
	rhsIndex := -1

	for i := range o.sortOrder {
		if o.sortOrder[i] == lhs.GroupVersion() {
			lhsIndex = i
		}
		if o.sortOrder[i] == rhs.GroupVersion() {
			rhsIndex = i
		}
	}

	if rhsIndex == -1 {
		return true
	}

	return lhsIndex < rhsIndex
}

type resourceByPreferredGroupVersion struct {
	list      []schema.GroupVersionResource
	sortOrder []schema.GroupVersion
}

func (o resourceByPreferredGroupVersion) Len() int      { return len(o.list) }
func (o resourceByPreferredGroupVersion) Swap(i, j int) { o.list[i], o.list[j] = o.list[j], o.list[i] }
func (o resourceByPreferredGroupVersion) Less(i, j int) bool {
	lhs := o.list[i]
	rhs := o.list[j]
	if lhs == rhs {
		return false
	}

	if lhs.GroupVersion() == rhs.GroupVersion() {
		return lhs.Resource < rhs.Resource
	}

	// otherwise, the difference is in the GroupVersion, so we need to sort with respect to the preferred order
	lhsIndex := -1
	rhsIndex := -1

	for i := range o.sortOrder {
		if o.sortOrder[i] == lhs.GroupVersion() {
			lhsIndex = i
		}
		if o.sortOrder[i] == rhs.GroupVersion() {
			rhsIndex = i
		}
	}

	if rhsIndex == -1 {
		return true
	}

	return lhsIndex < rhsIndex
}

// AmbiguousResourceError is returned if the RESTMapper finds multiple matches for a resource
type AmbiguousResourceError struct {
	PartialResource schema.GroupVersionKind

	MatchingResources []schema.GroupVersionResource
	MatchingKinds     []schema.GroupVersionKind
}

func (e *AmbiguousResourceError) Error() string {
	switch {
	case len(e.MatchingKinds) > 0 && len(e.MatchingResources) > 0:
		return fmt.Sprintf("%v matches multiple resources %v and kinds %v", e.PartialResource, e.MatchingResources, e.MatchingKinds)
	case len(e.MatchingKinds) > 0:
		return fmt.Sprintf("%v matches multiple kinds %v", e.PartialResource, e.MatchingKinds)
	case len(e.MatchingResources) > 0:
		return fmt.Sprintf("%v matches multiple resources %v", e.PartialResource, e.MatchingResources)
	}
	return fmt.Sprintf("%v matches multiple resources or kinds", e.PartialResource)
}
