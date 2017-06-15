package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceCodeDormantDatabase = "drmn"
	ResourceKindDormantDatabase = "DormantDatabase"
	ResourceNameDormantDatabase = "dormant-database"
	ResourceTypeDormantDatabase = "dormantdatabases"
)

type DormantDatabase struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DormantDatabaseSpec   `json:"spec,omitempty"`
	Status            DormantDatabaseStatus `json:"status,omitempty"`
}

type DormantDatabaseSpec struct {
	// If true, invoke wipe out operation
	// +optional
	WipeOut bool `json:"wipeOut,omitempty"`
	// If true, resumes database
	// +optional
	Resume bool `json:"resume,omitempty"`
	// Origin to store original database information
	Origin Origin `json:"origin,omitempty"`
}

type Origin struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Origin Spec to store original database Spec
	Spec OriginSpec `json:"spec,omitempty"`
}

type OriginSpec struct {
	// Elastic Spec
	// +optional
	Elastic *ElasticSpec `json:"elastic,omitempty"`
	// Postgres Spec
	// +optional
	Postgres *PostgresSpec `json:"postgres,omitempty"`
}

type DormantDatabasePhase string

const (
	// used for Databases that are paused
	DormantDatabasePhasePaused DormantDatabasePhase = "Paused"
	// used for Databases that are currently pausing
	DormantDatabasePhasePausing DormantDatabasePhase = "Pausing"
	// used for Databases that are wiped out
	DormantDatabasePhaseWipedOut DormantDatabasePhase = "WipedOut"
	// used for Databases that are currently wiping out
	DormantDatabasePhaseWipingOut DormantDatabasePhase = "WipingOut"
	// used for Databases that are currently recovering
	DormantDatabasePhaseResuming DormantDatabasePhase = "Resuming"
)

type DormantDatabaseStatus struct {
	CreationTime *metav1.Time         `json:"creationTime,omitempty"`
	PausingTime  *metav1.Time         `json:"pausingTime,omitempty"`
	WipeOutTime  *metav1.Time         `json:"wipeOutTime,omitempty"`
	Phase        DormantDatabasePhase `json:"phase,omitempty"`
	Reason       string               `json:"reason,omitempty"`
}

type DormantDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// Items is a list of DormantDatabase TPR objects
	Items []DormantDatabase `json:"items,omitempty"`
}
