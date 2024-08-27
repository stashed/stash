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

package v1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceID identifies a resource
type ResourceID struct {
	Group   string `json:"group" protobuf:"bytes,1,opt,name=group"`
	Version string `json:"version,omitempty" protobuf:"bytes,2,opt,name=version"`
	// Name is the plural name of the resource to serve.  It must match the name of the CustomResourceDefinition-registration
	// too: plural.group and it must be all lowercase.
	Name string `json:"name,omitempty" protobuf:"bytes,3,opt,name=name"`
	// Kind is the serialized kind of the resource.  It is normally CamelCase and singular.
	Kind  string        `json:"kind,omitempty" protobuf:"bytes,4,opt,name=kind"`
	Scope ResourceScope `json:"scope,omitempty" protobuf:"bytes,5,opt,name=scope,casttype=ResourceScope"`
}

// ResourceScope is an enum defining the different scopes available to a custom resource
type ResourceScope string

const (
	ClusterScoped   ResourceScope = "Cluster"
	NamespaceScoped ResourceScope = "Namespaced"
)

func (r ResourceID) GroupVersion() schema.GroupVersion {
	return schema.GroupVersion{Group: r.Group, Version: r.Version}
}

func (r ResourceID) GroupKind() schema.GroupKind {
	return schema.GroupKind{Group: r.Group, Kind: r.Kind}
}

func (r ResourceID) GroupResource() schema.GroupResource {
	return schema.GroupResource{Group: r.Group, Resource: r.Name}
}

func (r ResourceID) TypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{APIVersion: r.GroupVersion().String(), Kind: r.Kind}
}

func (r ResourceID) GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: r.Group, Version: r.Version, Resource: r.Name}
}

func (r ResourceID) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: r.Group, Version: r.Version, Kind: r.Kind}
}

func (r ResourceID) ListGroupVersionKind() schema.GroupVersionKind {
	kind := r.Kind + "List"
	if strings.HasSuffix(r.Kind, "List") {
		kind = r.Kind
	}
	return schema.GroupVersionKind{Group: r.Group, Version: r.Version, Kind: kind}
}

func (r ResourceID) MetaGVR() metav1.GroupVersionResource {
	return metav1.GroupVersionResource{Group: r.Group, Version: r.Version, Resource: r.Name}
}

func (r ResourceID) MetaGVK() metav1.GroupVersionKind {
	return metav1.GroupVersionKind{Group: r.Group, Version: r.Version, Kind: r.Kind}
}

func NewResourceID(mapping *meta.RESTMapping) *ResourceID {
	scope := ClusterScoped
	if mapping.Scope == meta.RESTScopeNamespace {
		scope = NamespaceScoped
	}
	return &ResourceID{
		Group:   mapping.Resource.Group,
		Version: mapping.Resource.Version,
		Name:    mapping.Resource.Resource,
		Kind:    mapping.GroupVersionKind.Kind,
		Scope:   scope,
	}
}

func FromMetaGVR(in metav1.GroupVersionResource) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    in.Group,
		Version:  in.Version,
		Resource: in.Resource,
	}
}

func ToMetaGVR(in schema.GroupVersionResource) metav1.GroupVersionResource {
	return metav1.GroupVersionResource{
		Group:    in.Group,
		Version:  in.Version,
		Resource: in.Resource,
	}
}

func FromMetaGVK(in metav1.GroupVersionKind) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   in.Group,
		Version: in.Version,
		Kind:    in.Kind,
	}
}

func ToMetaGVK(in schema.GroupVersionKind) metav1.GroupVersionKind {
	return metav1.GroupVersionKind{
		Group:   in.Group,
		Version: in.Version,
		Kind:    in.Kind,
	}
}

// FromAPIVersionAndKind returns a GVK representing the provided fields for types that
// do not use TypeMeta. This method exists to support test types and legacy serializations
// that have a distinct group and kind.
func FromAPIVersionAndKind(apiVersion, kind string) metav1.GroupVersionKind {
	if gv, err := schema.ParseGroupVersion(apiVersion); err == nil {
		return metav1.GroupVersionKind{Group: gv.Group, Version: gv.Version, Kind: kind}
	}
	return metav1.GroupVersionKind{Kind: kind}
}

func EqualsGVK(a schema.GroupVersionKind, b metav1.GroupVersionKind) bool {
	return a.Group == b.Group &&
		a.Version == b.Version &&
		a.Kind == b.Kind
}

func EqualsGVR(a schema.GroupVersionResource, b metav1.GroupVersionResource) bool {
	return a.Group == b.Group &&
		a.Version == b.Version &&
		a.Resource == b.Resource
}

func ExtractResourceID(mapper meta.RESTMapper, in ResourceID) (*ResourceID, error) {
	if in.Group == "core" {
		in.Group = ""
	}
	if in.Version != "" &&
		in.Kind != "" &&
		in.Name != "" &&
		in.Scope != "" {
		return &in, nil
	}

	kindFound := in.Kind != ""
	resFound := in.Name != ""
	if kindFound {
		if resFound {
			return &in, nil
		} else {
			var versions []string
			if in.Version != "" {
				versions = append(versions, in.Version)
			}
			mapping, err := mapper.RESTMapping(schema.GroupKind{
				Group: in.Group,
				Kind:  in.Kind,
			}, versions...)
			if err != nil {
				return nil, err
			}
			return NewResourceID(mapping), nil
		}
	} else {
		if resFound {
			gvk, err := mapper.KindFor(in.GroupVersionResource())
			if err != nil {
				return nil, err
			}
			mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				return nil, err
			}
			return NewResourceID(mapping), nil
		} else {
			return nil, fmt.Errorf("missing both Kind and Resource name for %+v", in)
		}
	}
}
