package v1alpha1

import (
	"github.com/appscode/go/encoding/json/types"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceKindBackupTrigger     = "BackupTrigger"
	ResourcePluralBackupTrigger   = "backupTriggers"
	ResourceSingularBackupTrigger = "backupTrigger"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupTrigger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BackupTriggerSpec   `json:"spec,omitempty"`
	Status            BackupTriggerStatus `json:"status,omitempty"`
}

type BackupTriggerSpec struct {
	// TargetBackup indicates the backup crd that will be triggered
	TargetBackup core.LocalObjectReference `json:"targetBackup"`
}

type BackupTriggerPhase string

const (
	BackupTriggerPending   BackupTriggerPhase = "Pending"
	BackupTriggerRunning   BackupTriggerPhase = "Running"
	BackupTriggerSucceeded BackupTriggerPhase = "Succeeded"
	BackupTriggerFailed    BackupTriggerPhase = "Failed"
	BackupTriggerUnknown   BackupTriggerPhase = "Unknown"
)

type BackupTriggerStatus struct {
	// observedGeneration is the most recent generation observed for this resource. It corresponds to the
	// resource's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration *types.IntHash     `json:"observedGeneration,omitempty"`
	Phase              BackupTriggerPhase `json:"phase,omitempty"`
	Stats              []TriggerStats     `json:"stats,omitempty"`
}

type TriggerStats struct {
	Host  string             `json:"host,omitempty"`
	Phase BackupTriggerPhase `json:"phase,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupTriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupTrigger `json:"items,omitempty"`
}
