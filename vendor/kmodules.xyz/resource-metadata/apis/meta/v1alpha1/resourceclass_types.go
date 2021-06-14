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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceKindResourceClass = "ResourceClass"
	ResourceResourceClass     = "resourceclass"
	ResourceResourceClasses   = "resourceclasses"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:skipVerbs=updateStatus
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=resourceclasses,singular=resourceclass
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ResourceClass struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ResourceClassSpec `json:"spec,omitempty"`
}

type ResourceClassSpec struct {
	ResourceClassInfo `json:",inline"`
	Weight            int     `json:"weight"`
	Entries           []Entry `json:"entries"`
}

type ResourceClassInfo struct {
	APIGroup string `json:"apiGroup,omitempty"`

	// Icons is an optional list of icons for an application. Icon information includes the source, size,
	// and mime type.
	Icons []ImageSpec `json:"icons,omitempty"`

	// Maintainers is an optional list of maintainers of the application. The maintainers in this list maintain the
	// the source code, images, and package for the application.
	Maintainers []ContactData `json:"maintainers,omitempty"`

	// Links are a list of descriptive URLs intended to be used to surface additional documentation, dashboards, etc.
	Links []Link `json:"links,omitempty"`
}

type Entry struct {
	Name string `json:"name"`
	// +optional
	Path string          `json:"path,omitempty"`
	Type *GroupResources `json:"type,omitempty"`
	// +optional
	Required bool `json:"required,omitempty"`
	// +optional
	Icons []ImageSpec `json:"icons,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

type ResourceClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceClass `json:"items,omitempty"`
}
