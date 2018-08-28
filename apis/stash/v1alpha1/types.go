package v1alpha1

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	store "kmodules.xyz/objectstore-api/api/v1"
)

const (
	ResourceKindRestic       = "Restic"
	ResourceSingularRestic   = "restic"
	ResourcePluralRestic     = "restics"
	ResourceKindRecovery     = "Recovery"
	ResourceSingularRecovery = "recovery"
	ResourcePluralRecovery   = "recoveries"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Restic struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ResticSpec `json:"spec,omitempty"`
}

type ResticSpec struct {
	Selector   metav1.LabelSelector `json:"selector,omitempty"`
	FileGroups []FileGroup          `json:"fileGroups,omitempty"`
	Backend    store.Backend        `json:"backend,omitempty"`
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
	Repository core.ObjectReference `json:"repository"`
	// Snapshot to recover. Default is latest snapshot.
	// +optional
	Snapshot         string                      `json:"snapshot,omitempty"`
	Paths            []string                    `json:"paths,omitempty"`
	RecoveredVolumes []store.LocalSpec           `json:"recoveredVolumes,omitempty"`
	ImagePullSecrets []core.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RecoveryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Recovery `json:"items,omitempty"`
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
	// observedGeneration is the most recent generation observed for this resource. It corresponds to the
	// resource's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// +optional
	ObservedGenerationHash string         `json:"observedGenerationHash,omitempty"`
	Phase                  RecoveryPhase  `json:"phase,omitempty"`
	Stats                  []RestoreStats `json:"stats,omitempty"`
}

type RestoreStats struct {
	Path     string        `json:"path,omitempty"`
	Phase    RecoveryPhase `json:"phase,omitempty"`
	Duration string        `json:"duration,omitempty"`
}

const (
	ResourceKindRepository     = "Repository"
	ResourcePluralRepository   = "repositories"
	ResourceSingularRepository = "repository"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RepositorySpec   `json:"spec,omitempty"`
	Status            RepositoryStatus `json:"status,omitempty"`
}

type RepositorySpec struct {
	Backend store.Backend `json:"backend,omitempty"`
	// If true, delete respective restic repository
	// +optional
	WipeOut bool `json:"wipeOut,omitempty"`
}

type RepositoryStatus struct {
	// observedGeneration is the most recent generation observed for this resource. It corresponds to the
	// resource's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// +optional
	ObservedGenerationHash   string       `json:"observedGenerationHash,omitempty"`
	FirstBackupTime          *metav1.Time `json:"firstBackupTime,omitempty"`
	LastBackupTime           *metav1.Time `json:"lastBackupTime,omitempty"`
	LastSuccessfulBackupTime *metav1.Time `json:"lastSuccessfulBackupTime,omitempty"`
	LastBackupDuration       string       `json:"lastBackupDuration,omitempty"`
	BackupCount              int64        `json:"backupCount,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items,omitempty"`
}
