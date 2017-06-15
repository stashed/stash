package api

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

const (
	ResourceCodeSnapshot = "snap"
	ResourceKindSnapshot = "Snapshot"
	ResourceNameSnapshot = "snapshot"
	ResourceTypeSnapshot = "snapshots"
)

type Snapshot struct {
	unversioned.TypeMeta `json:",inline,omitempty"`
	api.ObjectMeta       `json:"metadata,omitempty"`
	Spec                 SnapshotSpec   `json:"spec,omitempty"`
	Status               SnapshotStatus `json:"status,omitempty"`
}

type SnapshotSpec struct {
	// Database name
	DatabaseName string `json:"databaseName,omitempty"`
	// Snapshot Spec
	SnapshotStorageSpec `json:",inline,omitempty"`
}

type SnapshotPhase string

const (
	// used for Snapshots that are currently running
	SnapshotPhaseRunning SnapshotPhase = "Running"
	// used for Snapshots that are Succeeded
	SnapshotPhaseSuccessed SnapshotPhase = "Succeeded"
	// used for Snapshots that are Failed
	SnapshotPhaseFailed SnapshotPhase = "Failed"
)

type SnapshotStatus struct {
	StartTime      *unversioned.Time `json:"startTime,omitempty"`
	CompletionTime *unversioned.Time `json:"completionTime,omitempty"`
	Phase          SnapshotPhase     `json:"phase,omitempty"`
	Reason         string            `json:"reason,omitempty"`
}

type SnapshotList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	// Items is a list of Snapshot TPR objects
	Items []Snapshot `json:"items,omitempty"`
}
