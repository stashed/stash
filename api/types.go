package api

import (
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

type StatusCode int32
type RetentionStrategy string

const (
	// StatusUnknown indicates that a backup is in an uncertain state.
	StatusUnknown StatusCode = 0
	// StatusSuccess indicates that the last backup is successfull.
	StatusSuccess StatusCode = 1
	// StatusFailed indicates that the last backup is failed.
	StatusFailed StatusCode = 2
)

const (
	KeepLast    RetentionStrategy = "keep-last"
	KeepHourly  RetentionStrategy = "keep-hourly"
	KeepDaily   RetentionStrategy = "keep-daily"
	KeepWeekly  RetentionStrategy = "keep-weekly"
	KeepMonthly RetentionStrategy = "keep-monthly"
	KeepYearly  RetentionStrategy = "keep-yearly"
	KeepTag     RetentionStrategy = "keep-tag"
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
	// Tags of a snapshots
	Tags []string `json:"tags, omitempty"`
	// retention policy of snapshots
	BackupRetentionPolicy RetentionPolicy `json:"backupRetentionPolicy"`
	//  Some policy based garbage collection of old snapshots
	GarbageCollection string `json:"garbageCollection,omitempty"`
	//Secret where password of the restic is saved
	BackupSecretRef SecretRef `json:"backupSecretRef"`
}

type SecretRef struct {
	Name string `json:"name"`
	// key of restic password
	Key string `json:"key"`
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

type RetentionPolicy struct {
	Strategy          RetentionStrategy `json:"strategy"`
	SnapshotCount     int64             `json:"snapshotCount"`
	RetentionHostname string            `json:",retentionHostname,omitempty"`
	RetentionTags     []string          `json:"retentionTags,omitempty"`
	//To cleanup unreferenced data
	Prune bool `json:"prune,omitempty"`
}
