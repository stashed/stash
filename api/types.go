package api

import (
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

type RetentionStrategy string

const (
	KeepLast    RetentionStrategy = "keep-last"
	KeepHourly  RetentionStrategy = "keep-hourly"
	KeepDaily   RetentionStrategy = "keep-daily"
	KeepWeekly  RetentionStrategy = "keep-weekly"
	KeepMonthly RetentionStrategy = "keep-monthly"
	KeepYearly  RetentionStrategy = "keep-yearly"
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
	RetentionPolicy RetentionPolicy `json:"retentionPolicy"`
}

type BackupStatus struct {
	FirstBackupTime           time.Time `json:"firstBackupTime,omitempty"`
	LastBackupTime            time.Time `json:"lastBackupTime,omitempty"`
	LastSuccessfullBackupTime time.Time `json:"lastSuccessfullBackupTime,omitempty"`
	LastBackupDuration        float64   `json:"lastBackupDuration,omitempty"`
	BackupCount               int64     `json:"backupCount,omitempty"`
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
	Volume               api.Volume `json:"volume"`
	Path                 string     `json:"path"`
	RepositorySecretName string     `json:"repositorySecretName"`
}

type RetentionPolicy struct {
	Strategy       RetentionStrategy `json:"strategy,omitempty"`
	KeepTags       []string          `json:"keepTags,omitempty"`
	SnapshotCount  int64             `json:"snapshotCount,omitempty"`
	RetainHostname string            `json:",retainHostname,omitempty"`
	RetainTags     []string          `json:"retainTags,omitempty"`
	//To cleanup unreferenced data
	//Prune bool `json:"prune,omitempty"` //TODO not working for now.
}
