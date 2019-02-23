package v1beta1

import (
	"github.com/appscode/go/encoding/json/types"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceKindBackupSession     = "BackupSession"
	ResourceSingularBackupSession = "backupsession"
	ResourcePluralBackupSession   = "backupsessions"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupSession struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BackupSessionSpec   `json:"spec,omitempty"`
	Status            BackupSessionStatus `json:"status,omitempty"`
}

type BackupSessionSpec struct {
	// BackupConfiguration indicates the target BackupConfiguration crd
	BackupConfiguration core.LocalObjectReference `json:"backupConfiguration,omitempty"`
}

type BackupSessionPhase string

const (
	BackupSessionPending   BackupSessionPhase = "Pending"
	BackupSessionRunning   BackupSessionPhase = "Running"
	BackupSessionSucceeded BackupSessionPhase = "Succeeded"
	BackupSessionFailed    BackupSessionPhase = "Failed"
	BackupSessionUnknown   BackupSessionPhase = "Unknown"
)

type BackupSessionStatus struct {
	// ObservedGeneration is the most recent generation observed for this resource. It corresponds to the
	// resource's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration *types.IntHash `json:"observedGeneration,omitempty"`
	// Phase indicates the phase of the backup process for this BackupSession
	Phase BackupSessionPhase `json:"phase,omitempty"`
	// Stats shows statistics of this backup session
	// +optional
	Stats []BackupStats `json:"stats,omitempty"`
}

type BackupStats struct {
	// Directory indicates the directory that has been backed up in this session
	Directory string `json:"directory,omitempty"`
	// Snapshot indicates the name of the backup snapshot created in this backup session
	Snapshot string `json:"snapshot,omitempty"`
	// Size indicates the size of target data to backup
	Size string `json:"size,omitempty"`
	// Uploaded indicates size of data uploaded to backend in this backup session
	Uploaded string `json:"uploaded,omitempty"`
	// ProcessingTime indicates time taken to process the target data
	ProcessingTime string `json:"processingTime,omitempty"`
	// FileStats shows statistics of files of backup session
	FileStats FileStats `json:"fileStats,omitempty"`
}

type FileStats struct {
	// TotalFiles shows total number of files that has been backed up
	TotalFiles *int `json:"totalFiles,omitempty"`
	// NewFiles shows total number of new files that has been created since last backup
	NewFiles *int `json:"newFiles,omitempty"`
	// ModifiedFiles shows total number of files that has been modified since last backup
	ModifiedFiles *int `json:"modifiedFiles,omitempty"`
	// UnmodifiedFiles shows total number of files that has not been changed since last backup
	UnmodifiedFiles *int `json:"unmodifiedFiles,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupSessionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupSession `json:"items,omitempty"`
}
