package v1alpha1

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceKindContainerTemplate     = "ContainerTemplate"
	ResourcePluralContainerTemplate   = "containerTemplates"
	ResourceSingularContainerTemplate = "containerTemplate"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ContainerTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ContainerTemplateSpec `json:"spec,omitempty"`
}

type ContainerTemplateSpec struct {
	InitContainers []core.Container `json:"initContainers,omitempty"`
	Containers     []core.Container `json:"containers"`
	// List of volumes that can be mounted by containers belonging to the pod.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes
	// +optional
	Volumes []core.Volume `json:"volumes,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ContainerTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ContainerTemplate `json:"items,omitempty"`
}
