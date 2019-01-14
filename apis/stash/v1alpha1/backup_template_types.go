package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"
)

const (
	ResourceKindBackupTemplate     = "BackupTemplate"
	ResourcePluralBackupTemplate   = "backupTemplates"
	ResourceSingularBackupTemplate = "backupTemplate"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BackupTemplateSpec `json:"spec,omitempty"`
}

type BackupTemplateSpec struct {
	RepositorySpec      `json:",inline"`
	Type                BackupType `json:"type,omitempty"`
	Schedule            string     `json:"schedule,omitempty"`
	BackupAgent         string     `json:"backupAgent,omitempty"`
	RetentionPolicy     `json:"retentionPolicy,omitempty"`
	ContainerAttributes *core.Container `json:"containerAttributes,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use. For example,
	// in the case of docker, only DockerConfig type secrets are honored.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	ImagePullSecrets []core.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupTemplate `json:"items,omitempty"`
}
