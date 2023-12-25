/*
Copyright AppsCode Inc. and Contributors.

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
	"kmodules.xyz/resource-metrics/api"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const (
	ResourceKindGenericResource = "GenericResource"
	ResourceGenericResource     = "genericresource"
	ResourceGenericResources    = "genericresources"
)

// GenericResource is the Schema for any resource supported by resource-metrics library

// +genclient
// +genclient:onlyVerbs=get,list
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type GenericResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GenericResourceSpec   `json:"spec,omitempty"`
	Status *runtime.RawExtension `json:"status,omitempty"`
}

type GenericResourceSpec struct {
	// +optional
	Cluster kmapi.ClusterMetadata `json:"cluster,omitempty"`
	APIType kmapi.ResourceID      `json:"apiType"`
	Name    string                `json:"name"`
	// +optional
	UID types.UID `json:"uid,omitempty"`
	// +optional
	Version string `json:"version,omitempty"`
	// +optional
	Replicas int64 `json:"replicas,omitempty"`
	// +optional
	RoleReplicas api.ReplicaList `json:"roleReplicas,omitempty"`
	// +optional
	Mode string `json:"mode,omitempty"`
	// +optional
	TotalResource core.ResourceRequirements `json:"totalResource,omitempty"`
	// +optional
	AppResource core.ResourceRequirements `json:"appResource,omitempty"`
	// +optional
	RoleResourceLimits map[api.PodRole]core.ResourceList `json:"roleResourceLimits,omitempty"`
	// +optional
	RoleResourceRequests map[api.PodRole]core.ResourceList `json:"roleResourceRequests,omitempty"`
	Status               GenericResourceStatus             `json:"status"`
}

type GenericResourceStatus struct {
	// Status
	Status string `json:"status,omitempty"`
	// Message
	Message string `json:"message,omitempty"`
}

// GenericResourceList contains a list of GenericResource

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type GenericResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GenericResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GenericResource{}, &GenericResourceList{})
}
