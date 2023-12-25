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
	kmapi "kmodules.xyz/client-go/api/v1"
	"kmodules.xyz/resource-metadata/apis/shared"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceKindProject = "Project"
	ResourceProject     = "project"
	ResourceProjects    = "projects"
)

// ProjectSpec defines the desired state of Project
type ProjectSpec struct {
	// +kubebuilder:default=User
	Type              ProjectType            `json:"type,omitempty"`
	Namespaces        []string               `json:"namespaces,omitempty"`
	NamespaceSelector *metav1.LabelSelector  `json:"namespaceSelector,omitempty"`
	Monitoring        *ProjectMonitoring     `json:"monitoring,omitempty"`
	Presets           []shared.SourceLocator `json:"presets,omitempty"`
}

type ProjectMonitoring struct {
	PrometheusURL   string                 `json:"prometheusURL,omitempty"`
	GrafanaURL      string                 `json:"grafanaURL,omitempty"`
	AlertmanagerURL string                 `json:"alertmanagerURL,omitempty"`
	PrometheusRef   *kmapi.ObjectReference `json:"prometheusRef,omitempty"`
	AlertmanagerRef *kmapi.ObjectReference `json:"alertmanagerRef,omitempty"`
}

// +kubebuilder:validation:Enum=Default;System;User
type ProjectType string

const (
	ProjectDefault ProjectType = "Default"
	ProjectSystem  ProjectType = "System"
	ProjectUser    ProjectType = "User"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:onlyVerbs=get,list
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster

// Project is the Schema for the projects API
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ProjectSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
