package v1alpha1

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceKindBackup     = "Backup"
	ResourceSingularBackup = "backup"
	ResourcePluralBackup   = "backups"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Backup struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BackupSpec `json:"spec,omitempty"`
}

type BackupSpec struct {
	// Type indicates weather backup is Scheduled or OneTime
	Type     BackupType `json:"type,omitempty"`
	Schedule string     `json:"schedule,omitempty"`
	// BackupAgent specify the ContainerTemplate that will be used for backup sidecar or job
	BackupAgent string `json:"backupAgent,omitempty"`
	// Repository refer to the Repository crd that hold backend information
	Repository core.LocalObjectReference `json:"repository"`
	// TargetRef specify the backup target
	TargetRef core.ObjectReference `json:"targetRef"`
	// TargetDirectories specify the directories to backup when the target is a volume
	//+optional
	TargetDirectories []string `json:"targetDirectories,omitempty"`
	// RetentionPolicy indicates the policy to follow to clean old backup snapshots
	RetentionPolicy `json:"retentionPolicy,omitempty"`
	//Indicates that the Backup is paused from taking backup. Default value is 'false'
	// +optional
	Paused bool `json:"paused,omitempty"`
	//ContainerAttributes allow to specify Env, Resources, SecurityContext etc. for backup sidecar or job's container
	//+optional
	ContainerAttributes *core.Container `json:"containerAttributes,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use. For example,
	// in the case of docker, only DockerConfig type secrets are honored.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	ImagePullSecrets []core.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Backup `json:"items,omitempty"`
}

type BackupType string

const (
	ScheduledBackup BackupType = "Scheduled" // default, backup using sidecar or cron job
	OneTimeBackup   BackupType = "OneTime"   // backup using init container or job
)

type RetentionStrategy string

const (
	KeepLast    RetentionStrategy = "--keep-last"
	KeepHourly  RetentionStrategy = "--keep-hourly"
	KeepDaily   RetentionStrategy = "--keep-daily"
	KeepWeekly  RetentionStrategy = "--keep-weekly"
	KeepMonthly RetentionStrategy = "--keep-monthly"
	KeepYearly  RetentionStrategy = "--keep-yearly"
	KeepTag     RetentionStrategy = "--keep-tag"
)

type RetentionPolicy struct {
	Name        string   `json:"name,omitempty"`
	KeepLast    int      `json:"keepLast,omitempty"`
	KeepHourly  int      `json:"keepHourly,omitempty"`
	KeepDaily   int      `json:"keepDaily,omitempty"`
	KeepWeekly  int      `json:"keepWeekly,omitempty"`
	KeepMonthly int      `json:"keepMonthly,omitempty"`
	KeepYearly  int      `json:"keepYearly,omitempty"`
	KeepTags    []string `json:"keepTags,omitempty"`
	Prune       bool     `json:"prune,omitempty"`
	DryRun      bool     `json:"dryRun,omitempty"`
}
