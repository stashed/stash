package v1alpha1

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceKindRestic   = "Restic"
	ResourceNameRestic   = "restic"
	ResourceTypeRestic   = "restics"
	ResourceKindRecovery = "Recovery"
	ResourceNameRecovery = "recovery"
	ResourceTypeRecovery = "recoveries"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Restic struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ResticSpec   `json:"spec,omitempty"`
	Status            ResticStatus `json:"status,omitempty"`
}

type ResticSpec struct {
	Selector   metav1.LabelSelector `json:"selector,omitempty"`
	FileGroups []FileGroup          `json:"fileGroups,omitempty"`
	Backend    Backend              `json:"backend,omitempty"`
	Schedule   string               `json:"schedule,omitempty"`
	// Pod volumes to mount into the sidecar container's filesystem.
	VolumeMounts []core.VolumeMount `json:"volumeMounts,omitempty"`
	// Compute Resources required by the sidecar container.
	Resources         core.ResourceRequirements `json:"resources,omitempty"`
	RetentionPolicies []RetentionPolicy         `json:"retentionPolicies,omitempty"`
	// https://github.com/appscode/stash/issues/225
	Type BackupType `json:"type,omitempty"`
	//Indicates that the Restic is paused from taking backup. Default value is 'false'
	// +optional
	Paused bool `json:"paused,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use. For example,
	// in the case of docker, only DockerConfig type secrets are honored.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	ImagePullSecrets []core.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

type ResticStatus struct {
	FirstBackupTime          *metav1.Time `json:"firstBackupTime,omitempty"`
	LastBackupTime           *metav1.Time `json:"lastBackupTime,omitempty"`
	LastSuccessfulBackupTime *metav1.Time `json:"lastSuccessfulBackupTime,omitempty"`
	LastBackupDuration       string       `json:"lastBackupDuration,omitempty"`
	BackupCount              int64        `json:"backupCount,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ResticList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Restic `json:"items,omitempty"`
}

type FileGroup struct {
	// Source of the backup volumeName:path
	Path string `json:"path,omitempty"`
	// Tags of a snapshots
	Tags []string `json:"tags,omitempty"`
	// retention policy of snapshots
	RetentionPolicyName string `json:"retentionPolicyName,omitempty"`
}

type Backend struct {
	StorageSecretName string `json:"storageSecretName,omitempty"`

	Local *LocalSpec `json:"local,omitempty"`
	S3    *S3Spec    `json:"s3,omitempty"`
	GCS   *GCSSpec   `json:"gcs,omitempty"`
	Azure *AzureSpec `json:"azure,omitempty"`
	Swift *SwiftSpec `json:"swift,omitempty"`
	B2    *B2Spec    `json:"b2,omitempty"`
	// Rest  *RestServerSpec `json:"rest,omitempty"`
}

type LocalSpec struct {
	core.VolumeSource `json:",inline"`
	MountPath         string `json:"mountPath,omitempty"`
	SubPath           string `json:"subPath,omitempty"`
}

type S3Spec struct {
	Endpoint string `json:"endpoint,omitempty"`
	Bucket   string `json:"bucket,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
}

type GCSSpec struct {
	Bucket string `json:"bucket,omitempty"`
	Prefix string `json:"prefix,omitempty"`
}

type AzureSpec struct {
	Container string `json:"container,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
}

type SwiftSpec struct {
	Container string `json:"container,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
}

type B2Spec struct {
	Bucket string `json:"bucket,omitempty"`
	Prefix string `json:"prefix,omitempty"`
}

type RestServerSpec struct {
	URL string `json:"url,omitempty"`
}

type BackupType string

const (
	BackupOnline  BackupType = "online"  // default, injects sidecar
	BackupOffline BackupType = "offline" // injects init container
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

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Recovery struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RecoverySpec   `json:"spec,omitempty"`
	Status            RecoveryStatus `json:"status,omitempty"`
}

type RecoverySpec struct {
	Backend          Backend                     `json:"backend,omitempty"`
	Paths            []string                    `json:"paths,omitempty"`
	Workload         LocalTypedReference         `json:"workload,omitempty"`
	PodOrdinal       string                      `json:"podOrdinal,omitempty"`
	NodeName         string                      `json:"nodeName,omitempty"`
	RecoveredVolumes []LocalSpec                 `json:"recoveredVolumes,omitempty"`
	ImagePullSecrets []core.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RecoveryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Recovery `json:"items,omitempty"`
}

// LocalTypedReference contains enough information to let you inspect or modify the referred object.
type LocalTypedReference struct {
	// Kind of the referent.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	Kind string `json:"kind,omitempty"`
	// Name of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	// +optional
	Name string `json:"name,omitempty"`
	// API version of the referent.
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}

type RecoveryPhase string

const (
	RecoveryPending   RecoveryPhase = "Pending"
	RecoveryRunning   RecoveryPhase = "Running"
	RecoverySucceeded RecoveryPhase = "Succeeded"
	RecoveryFailed    RecoveryPhase = "Failed"
	RecoveryUnknown   RecoveryPhase = "Unknown"
)

type RecoveryStatus struct {
	Phase RecoveryPhase  `json:"phase,omitempty"`
	Stats []RestoreStats `json:"stats,omitempty"`
}

type RestoreStats struct {
	Path     string        `json:"path,omitempty"`
	Phase    RecoveryPhase `json:"phase,omitempty"`
	Duration string        `json:"duration,omitempty"`
}
