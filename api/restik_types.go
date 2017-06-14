package api

import (
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

type Restik struct {
	unversioned.TypeMeta `json:",inline,omitempty"`
	api.ObjectMeta       `json:"metadata,omitempty"`
	Spec                 RestikSpec   `json:"spec,omitempty"`
	Status               RestikStatus `json:"status,omitempty"`
}

type RestikSpec struct {
	// Source of the backup volumeName:path
	Source Source `json:"source"`
	// Destination of the backup
	Destination Destination `json:"destination"`
	// How frequently restic command will be run
	Schedule string `json:"schedule"`
	// Tags of a snapshots
	Tags []string `json:"tags,omitempty"`
	// retention policy of snapshots
	RetentionPolicy RetentionPolicy `json:"retentionPolicy,omitempty"`
}

type RestikStatus struct {
	FirstBackupTime          *unversioned.Time `json:"firstBackupTime,omitempty"`
	LastBackupTime           *unversioned.Time `json:"lastBackupTime,omitempty"`
	LastSuccessfulBackupTime *unversioned.Time `json:"lastSuccessfulBackupTime,omitempty"`
	LastBackupDuration       string            `json:"lastBackupDuration,omitempty"`
	BackupCount              int64             `json:"backupCount,omitempty"`
}

type RestikList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	Items                []Restik `json:"items,omitempty"`
}

type Source struct {
	VolumeName string `json:"volumeName"`
	Path       string `json:"path"`
}

type Destination struct {
	Volume               api.Volume `json:"volume"`
	Path                 string     `json:"path"`
	RepositorySecretName string     `json:"repositorySecretName"`
}

type RetentionPolicy struct {
	KeepLastSnapshots    int      `json:"keepLastSnapshots,omitempty"`
	KeepHourlySnapshots  int      `json:"keepHourlySnapshots,omitempty"`
	KeepDailySnapshots   int      `json:"keepDailySnapshots,omitempty"`
	KeepWeeklySnapshots  int      `json:"keepWeeklySnapshots,omitempty"`
	KeepMonthlySnapshots int      `json:"keepMonthlySnapshots,omitempty"`
	KeepYearlySnapshots  int      `json:"keepYearlySnapshots,omitempty"`
	KeepTags             []string `json:"keepTags,omitempty"`
	RetainHostname       string   `json:"retainHostname,omitempty"`
	RetainTags           []string `json:"retainTags,omitempty"`
}
