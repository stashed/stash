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

//go:generate go-enum --mustparse --names --values
package v1

import (
	"fmt"
	"strings"
	"unicode"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TypedObjectReference represents an typed namespaced object.
type TypedObjectReference struct {
	APIGroup string `json:"apiGroup,omitempty" protobuf:"bytes,1,opt,name=apiGroup"`
	Kind     string `json:"kind,omitempty" protobuf:"bytes,2,opt,name=kind"`
	// Namespace of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
	// Name of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	Name string `json:"name" protobuf:"bytes,4,opt,name=name"`
}

// ObjectReference contains enough information to let you inspect or modify the referred object.
type ObjectReference struct {
	// Namespace of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,1,opt,name=namespace"`
	// Name of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
}

// WithNamespace sets the namespace if original namespace is empty.
// Never changes the original ObjectReference.
func (ref *ObjectReference) WithNamespace(fallback string) *ObjectReference {
	if ref == nil {
		return nil
	}
	if ref.Namespace != "" {
		return ref
	}
	out := *ref
	out.Namespace = fallback
	return &out
}

func (ref ObjectReference) ObjectKey() client.ObjectKey {
	return client.ObjectKey{Namespace: ref.Namespace, Name: ref.Name}
}

type OID string

type ObjectID struct {
	Group     string `json:"group,omitempty" protobuf:"bytes,1,opt,name=group"`
	Kind      string `json:"kind,omitempty" protobuf:"bytes,2,opt,name=kind"`
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
	Name      string `json:"name,omitempty" protobuf:"bytes,4,opt,name=name"`
}

func (oid *ObjectID) OID() OID {
	return OID(fmt.Sprintf("G=%s,K=%s,NS=%s,N=%s", oid.Group, oid.Kind, oid.Namespace, oid.Name))
}

// WithNamespace sets the namespace if original namespace is empty.
// Never changes the original ObjectID.
func (oid *ObjectID) WithNamespace(fallback string) *ObjectID {
	if oid == nil {
		return nil
	}
	if oid.Namespace != "" {
		return oid
	}
	out := *oid
	out.Namespace = fallback
	return &out
}

func NewObjectID(obj client.Object) *ObjectID {
	gvk := obj.GetObjectKind().GroupVersionKind()
	return &ObjectID{
		Group:     gvk.Group,
		Kind:      gvk.Kind,
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

func ParseObjectID(key OID) (*ObjectID, error) {
	var id ObjectID

	chunks := strings.Split(string(key), ",")
	for _, chunk := range chunks {
		parts := strings.FieldsFunc(chunk, func(r rune) bool {
			return r == '=' || unicode.IsSpace(r)
		})
		if len(parts) == 0 || len(parts) > 2 {
			return nil, fmt.Errorf("invalid chunk %s", chunk)
		}

		switch parts[0] {
		case "G":
			if len(parts) == 2 {
				id.Group = parts[1]
			}
		case "K":
			if len(parts) == 1 {
				return nil, fmt.Errorf("kind not set")
			}
			id.Kind = parts[1]
		case "NS":
			if len(parts) == 2 {
				id.Namespace = parts[1]
			}
		case "N":
			if len(parts) == 1 {
				return nil, fmt.Errorf("name not set")
			}
			id.Name = parts[1]
		default:
			return nil, fmt.Errorf("unknown key %s", parts[0])
		}
	}
	return &id, nil
}

func MustParseObjectID(key OID) *ObjectID {
	oid, err := ParseObjectID(key)
	if err != nil {
		panic(err)
	}
	return oid
}

func ObjectIDMap(key OID) (map[string]interface{}, error) {
	id := map[string]interface{}{
		"group":     "",
		"kind":      "",
		"namespace": "",
		"name":      "",
	}

	chunks := strings.Split(string(key), ",")
	for _, chunk := range chunks {
		parts := strings.FieldsFunc(chunk, func(r rune) bool {
			return r == '=' || unicode.IsSpace(r)
		})
		if len(parts) == 0 || len(parts) > 2 {
			return nil, fmt.Errorf("invalid chunk %s", chunk)
		}

		switch parts[0] {
		case "G":
			if len(parts) == 2 {
				id["group"] = parts[1]
			}
		case "K":
			if len(parts) == 1 {
				return nil, fmt.Errorf("kind not set")
			}
			id["kind"] = parts[1]
		case "NS":
			if len(parts) == 2 {
				id["namespace"] = parts[1]
			}
		case "N":
			if len(parts) == 1 {
				return nil, fmt.Errorf("name not set")
			}
			id["name"] = parts[1]
		default:
			return nil, fmt.Errorf("unknown key %s", parts[0])
		}
	}
	return id, nil
}

func (oid *ObjectID) GroupKind() schema.GroupKind {
	return schema.GroupKind{Group: oid.Group, Kind: oid.Kind}
}

func (oid *ObjectID) MetaGroupKind() metav1.GroupKind {
	return metav1.GroupKind{Group: oid.Group, Kind: oid.Kind}
}

func (oid *ObjectID) ObjectReference() ObjectReference {
	return ObjectReference{Namespace: oid.Namespace, Name: oid.Name}
}

func (oid *ObjectID) ObjectKey() client.ObjectKey {
	return client.ObjectKey{Namespace: oid.Namespace, Name: oid.Name}
}

type ObjectInfo struct {
	Resource ResourceID      `json:"resource" protobuf:"bytes,1,opt,name=resource"`
	Ref      ObjectReference `json:"ref" protobuf:"bytes,2,opt,name=ref"`
}

// +kubebuilder:validation:Enum=authn;authz;auth_secret;backup_via;catalog;cert_issuer;config;connect_via;exposed_by;event;located_on;monitored_by;ocm_bind;offshoot;ops;placed_into;policy;recommended_for;restore_into;scaled_by;source;storage;view
// ENUM(authn,authz,auth_secret,backup_via,catalog,cert_issuer,config,connect_via,exposed_by,event,located_on,monitored_by,ocm_bind,offshoot,ops,placed_into,policy,recommended_for,restore_into,scaled_by,source,storage,view)
type EdgeLabel string

func (e EdgeLabel) Direct() bool {
	return e == EdgeLabelOffshoot ||
		e == EdgeLabelView ||
		e == EdgeLabelOps ||
		e == EdgeLabelRecommendedFor
}
