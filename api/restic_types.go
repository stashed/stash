package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
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

type Restic struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ResticSpec   `json:"spec,omitempty"`
	Status            ResticStatus `json:"status,omitempty"`
}

type ResticSpec struct {
	// Selector is a label query over a set of resources, in this case pods.
	// Required.
	Selector metav1.LabelSelector `json:"selector,omitempty"`
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

type ResticStatus struct {
	FirstBackupTime          *metav1.Time `json:"firstBackupTime,omitempty"`
	LastBackupTime           *metav1.Time `json:"lastBackupTime,omitempty"`
	LastSuccessfulBackupTime *metav1.Time `json:"lastSuccessfulBackupTime,omitempty"`
	LastBackupDuration       string       `json:"lastBackupDuration,omitempty"`
	BackupCount              int64        `json:"backupCount,omitempty"`
}

type ResticList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Restic `json:"items,omitempty"`
}

type Source struct {
	VolumeName string `json:"volumeName"`
	Path       string `json:"path"`
}

type Destination struct {
	Volume               apiv1.Volume `json:"volume"`
	Path                 string       `json:"path"`
	RepositorySecretName string       `json:"repositorySecretName"`
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
