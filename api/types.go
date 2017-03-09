package kube

import (
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

type Status_Code int32

const (
	// Status_UNKNOWN indicates that a backup is in an uncertain state.
	Status_UNKNOWN Status_Code = 0
	// Status_DEPLOYED indicates that the last backup is successfull.
	Status_Success Status_Code = 1
	// Status_DELETED indicates that the last backup is failed.
	Status_Failed Status_Code = 2
)

type Backup struct {
	unversioned.TypeMeta `json:",inline,omitempty"`
	api.ObjectMeta       `json:"metadata,omitempty"`
	Spec                 BackupSpec   `json:"spec,omitempty"`
	Status               BackupStatus `json:"status,omitempty"`
}

type BackupSpec struct {
	// Source of the backup volumename:path
	Source backupSource `json:"backupSource"`
	// Destination of the backup
	Destination backupDestination `json:"destination"`
	// How frequently backup command will be run
	Schedule string `json:"schedule"`
	//  Some policy based garbage collection of old snapshots
	GarbageCollection string `json:"garbageCollection,omitempty"`
}

type BackupStatus struct {
	LastBackupStatus      Status_Code `json:"lastBackupStatus"`
	Created               time.Time   `json:"created,omitempty"`
	LastBackup            time.Time   `json:"lastBackup,omitempty"`
	LastSuccessfullBackup time.Time   `json:"lastSuccessfullBackup"`
	Message               string      `json:"message"`
}

type BackupList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	Items                []Backup `json:"items,omitempty"`
}

type backupSource struct {
	VolumeName string `json:"volumeName"`
	Path       string `json:"path"`
}

type backupDestination struct {
	Volume api.Volume `json:"volume"`
	Path   string     `json:"path"`
}
