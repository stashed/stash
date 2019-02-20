package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceKindBackupTemplate     = "BackupTemplate"
	ResourcePluralBackupTemplate   = "backuptemplates"
	ResourceSingularBackupTemplate = "backuptemplate"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BackupTemplateSpec `json:"spec,omitempty"`
}

type BackupTemplateSpec struct {
	// RepositorySpec is used to create Repository crd for respective workload
	RepositorySpec `json:",inline"`
	Schedule       string `json:"schedule,omitempty"`
	// Task specify the Task crd that specifies steps for backup process
	// +optional
	Task TaskRef `json:"task,omitempty"`
	// RetentionPolicy indicates the policy to follow to clean old backup snapshots
	RetentionPolicy `json:"retentionPolicy,omitempty"`
	// ExecutionEnvironment allow to specify Resources, NodeSelector, Affinity, Toleration, ReadinessProbe etc.
	//+optional
	ExecutionEnvironment ExecutionEnvironment `json:"executionEnvironment,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupTemplate `json:"items,omitempty"`
}
