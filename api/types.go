package kube

import (
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

type StatusCode int32

const (
	// StatusUnknown indicates that a backup is in an uncertain state.
	StatusUnknown StatusCode = 0
	// StatusSuccess indicates that the last backup is successfull.
	StatusSuccess StatusCode = 1
	// StatusFailed indicates that the last backup is failed.
	StatusFailed StatusCode = 2
)

type Backup struct {
	unversioned.TypeMeta `json:",inline,omitempty"`
	api.ObjectMeta       `json:"metadata,omitempty"`
	Spec                 BackupSpec   `json:"spec,omitempty"`
	Status               BackupStatus `json:"status,omitempty"`
}

type BackupSpec struct {
	// Source of the backup volumename:path
	Source BackupSource `json:"backupSource"`
	// Destination of the backup
	Destination BackupDestination `json:"destination"`
	// How frequently backup command will be run
	Schedule string `json:"schedule"`
	//  Some policy based garbage collection of old snapshots
	GarbageCollection string `json:"garbageCollection,omitempty"`
}

type BackupStatus struct {
	LastBackupStatus      StatusCode `json:"lastBackupStatus"`
	Created               time.Time  `json:"created,omitempty"`
	LastBackup            time.Time  `json:"lastBackup,omitempty"`
	LastSuccessfullBackup time.Time  `json:"lastSuccessfullBackup"`
	Message               string     `json:"message"`
	BackupCount           int64      `json:"backupCount"`
}

type BackupList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	Items                []Backup `json:"items,omitempty"`
}

type BackupSource struct {
	VolumeName string `json:"volumeName"`
	Path       string `json:"path"`
}

type BackupDestination struct {
	Volume api.Volume `json:"volume"`
	Path   string     `json:"path"`
}
