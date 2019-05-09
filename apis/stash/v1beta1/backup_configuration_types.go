package v1beta1

import (
	core "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
)

const (
	ResourceKindBackupConfiguration     = "BackupConfiguration"
	ResourceSingularBackupConfiguration = "backupconfiguration"
	ResourcePluralBackupConfiguration   = "backupconfigurations"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupConfiguration struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BackupConfigurationSpec `json:"spec,omitempty"`
}

type BackupConfigurationSpec struct {
	Schedule string `json:"schedule,omitempty"`
	// Driver indicates the name of the agent to use to backup the target.
	// Supported values are "Restic", "VolumeSnapshotter".
	// Default value is "Restic".
	// +optional
	Driver Snapshotter `json:"driver,omitempty"`
	// Repository refer to the Repository crd that holds backend information
	// +optional
	Repository core.LocalObjectReference `json:"repository,omitempty"`
	// Task specify the Task crd that specifies the steps to take backup
	// +optional
	Task TaskRef `json:"task,omitempty"`
	// Target specify the backup target
	// +optional
	Target *BackupTarget `json:"target,omitempty"`
	// RetentionPolicy indicates the policy to follow to clean old backup snapshots
	RetentionPolicy v1alpha1.RetentionPolicy `json:"retentionPolicy,omitempty"`
	// Indicates that the BackupConfiguration is paused from taking backup. Default value is 'false'
	// +optional
	Paused bool `json:"paused,omitempty"`
	// RuntimeSettings allow to specify Resources, NodeSelector, Affinity, Toleration, ReadinessProbe etc.
	//+optional
	RuntimeSettings ofst.RuntimeSettings `json:"runtimeSettings,omitempty"`
	// Temp directory configuration for functions/sidecar
	// An `EmptyDir` will always be mounted at /tmp with this settings
	//+optional
	TempDir EmptyDirSettings `json:"tempDir,omitempty"`
}

type EmptyDirSettings struct {
	Medium    core.StorageMedium `json:"medium,omitempty"`
	SizeLimit *resource.Quantity `json:"sizeLimit,omitempty"`
	// More info: https://github.com/restic/restic/blob/master/doc/manual_rest.rst#caching
	DisableCaching bool `json:"disableCaching,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupConfiguration `json:"items,omitempty"`
}

type Snapshotter string

const (
	ResticSnapshotter Snapshotter = "Restic"
	VolumeSnapshotter Snapshotter = "VolumeSnapshotter"
)
