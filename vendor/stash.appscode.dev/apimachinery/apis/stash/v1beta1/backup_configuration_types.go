/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
	prober "kmodules.xyz/prober/api/v1"
)

const (
	ResourceKindBackupConfiguration     = "BackupConfiguration"
	ResourceSingularBackupConfiguration = "backupconfiguration"
	ResourcePluralBackupConfiguration   = "backupconfigurations"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=backupconfigurations,singular=backupconfiguration,shortName=bc,categories={stash,appscode,all}
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Task",type="string",JSONPath=".spec.task.name"
// +kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=".spec.schedule"
// +kubebuilder:printcolumn:name="Paused",type="boolean",JSONPath=".spec.paused"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type BackupConfiguration struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BackupConfigurationSpec   `json:"spec,omitempty"`
	Status            BackupConfigurationStatus `json:"status,omitempty"`
}

type BackupConfigurationTemplateSpec struct {
	// Task specify the Task crd that specifies the steps to take backup
	// +optional
	Task TaskRef `json:"task,omitempty"`
	// Target specify the backup target
	// +optional
	Target *BackupTarget `json:"target,omitempty"`
	// RuntimeSettings allow to specify Resources, NodeSelector, Affinity, Toleration, ReadinessProbe etc.
	// +optional
	RuntimeSettings ofst.RuntimeSettings `json:"runtimeSettings,omitempty"`
	// Temp directory configuration for functions/sidecar
	// An `EmptyDir` will always be mounted at /tmp with this settings
	// +optional
	TempDir EmptyDirSettings `json:"tempDir,omitempty"`
	// InterimVolumeTemplate specifies a template for a volume to hold targeted data temporarily
	// before uploading to backend or inserting into target. It is only usable for job model.
	// Don't specify it in sidecar model.
	// +optional
	InterimVolumeTemplate *ofst.PersistentVolumeClaim `json:"interimVolumeTemplate,omitempty"`
	// Actions that Stash should take in response to backup sessions.
	// +optional
	Hooks *BackupHooks `json:"hooks,omitempty"`
}

type BackupConfigurationSpec struct {
	BackupConfigurationTemplateSpec `json:",inline,omitempty"`
	// Schedule specifies the schedule for invoking backup sessions
	// +optional
	Schedule string `json:"schedule,omitempty"`
	// Driver indicates the name of the agent to use to backup the target.
	// Supported values are "Restic", "VolumeSnapshotter".
	// Default value is "Restic".
	// +optional
	// +kubebuilder:default=Restic
	Driver Snapshotter `json:"driver,omitempty"`
	// Repository refer to the Repository crd that holds backend information
	// +optional
	Repository kmapi.ObjectReference `json:"repository,omitempty"`
	// RetentionPolicy indicates the policy to follow to clean old backup snapshots
	RetentionPolicy v1alpha1.RetentionPolicy `json:"retentionPolicy"`
	// Indicates that the BackupConfiguration is paused from taking backup. Default value is 'false'
	// +optional
	Paused bool `json:"paused,omitempty"`
	// BackupHistoryLimit specifies the number of BackupSession and it's associate resources to keep.
	// This is helpful for debugging purpose.
	// Default: 1
	// +optional
	BackupHistoryLimit *int32 `json:"backupHistoryLimit,omitempty"`
	// TimeOut specifies the maximum duration of backup. BackupSession will be considered Failed
	// if backup does not complete within this time limit. By default, Stash don't set any timeout for backup.
	// +optional
	TimeOut string `json:"timeOut,omitempty"`
}

// Hooks describes actions that Stash should take in response to backup sessions. For the PostBackup
// and PreBackup handlers, backup process blocks until the action is complete,
// unless the container process fails, in which case the handler is aborted.
type BackupHooks struct {
	// PreBackup is called immediately before a backup session is initiated.
	// +optional
	PreBackup *prober.Handler `json:"preBackup,omitempty"`

	// PostBackup is called immediately after a backup session is complete.
	// +optional
	PostBackup *prober.Handler `json:"postBackup,omitempty"`
}

type EmptyDirSettings struct {
	Medium    core.StorageMedium `json:"medium,omitempty"`
	SizeLimit *resource.Quantity `json:"sizeLimit,omitempty"`
	// More info: https://github.com/restic/restic/blob/master/doc/manual_rest.rst#caching
	DisableCaching bool `json:"disableCaching,omitempty"`
}

// +kubebuilder:validation:Enum=Restic;VolumeSnapshotter
type Snapshotter string

const (
	ResticSnapshotter Snapshotter = "Restic"
	VolumeSnapshotter Snapshotter = "VolumeSnapshotter"
)

type BackupConfigurationStatus struct {
	// ObservedGeneration is the most recent generation observed for this BackupConfiguration. It corresponds to the
	// BackupConfiguration's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Conditions shows current backup setup condition of the BackupConfiguration.
	// +optional
	Conditions []kmapi.Condition `json:"conditions,omitempty"`
	// Phase indicates phase of this BackupConfiguration.
	// +optional
	Phase BackupInvokerPhase `json:"phase,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupConfiguration `json:"items,omitempty"`
}

// +kubebuilder:validation:Enum=Invalid;Ready;NotReady
type BackupInvokerPhase string

const (
	BackupInvokerInvalid  BackupInvokerPhase = "Invalid"
	BackupInvokerReady    BackupInvokerPhase = "Ready"
	BackupInvokerNotReady BackupInvokerPhase = "NotReady"
)

// ==================== Condition Types ============================
const (
	// BackupTargetFound indicates whether the backup target was found
	BackupTargetFound = "BackupTargetFound"

	// StashSidecarInjected indicates whether stash sidecar was injected into the targeted workload
	// This condition is applicable only for sidecar model
	StashSidecarInjected = "StashSidecarInjected"

	// CronJobCreated indicates whether the backup triggering CronJob was created
	CronJobCreated = "CronJobCreated"

	// RepositoryFound indicates whether the respective Repository object was found or not.
	RepositoryFound = "RepositoryFound"

	// BackendSecretFound indicates whether the respective backend secret was found or not.
	BackendSecretFound = "BackendSecretFound"

	// ValidationPassed indicates the validation conditions of the CRD are passed or not.
	ValidationPassed = "ValidationPassed"
)

// ======================= Condition Reasons ===========================
const (
	// TargetAvailable indicates that the condition transitioned to this state because the target was available
	TargetAvailable = "TargetAvailable"
	// TargetNotAvailable indicates that the condition transitioned to this state because the target was not available
	TargetNotAvailable = "TargetNotAvailable"
	// UnableToCheckTargetAvailability indicates that the condition transitioned to this state because operator was unable
	// to check the target availability
	UnableToCheckTargetAvailability = "UnableToCheckTargetAvailability"

	// SidecarInjectionSucceeded indicates that the condition transitioned to this state because sidecar was injected
	// successfully into the targeted workload
	SidecarInjectionSucceeded = "SidecarInjectionSucceeded"
	// SidecarInjectionFailed indicates that the condition transitioned to this state because operator was unable
	// to inject sidecar into the targeted workload
	SidecarInjectionFailed = "SidecarInjectionFailed"

	// CronJobCreationSucceeded indicates that the condition transitioned to this state because backup triggering CronJob was created successfully
	CronJobCreationSucceeded = "CronJobCreationSucceeded"
	// CronJobCreationFailed indicates that the condition transitioned to this state because operator was unable to create backup triggering CronJob
	CronJobCreationFailed = "CronJobCreationFailed"

	// RepositoryAvailable indicates that the condition transitioned to this state because the Repository was available
	RepositoryAvailable = "RepositoryAvailable"
	// RepositoryNotAvailable indicates that the condition transitioned to this state because the Repository was not available
	RepositoryNotAvailable = "RepositoryNotAvailable"
	// UnableToCheckRepositoryAvailability indicates that the condition transitioned to this state because operator was unable
	// to check the Repository availability
	UnableToCheckRepositoryAvailability = "UnableToCheckRepositoryAvailability"

	// BackendSecretAvailable indicates that the condition transitioned to this state because the backend Secret was available
	BackendSecretAvailable = "BackendSecretAvailable"
	// BackendSecretNotAvailable indicates that the condition transitioned to this state because the backend Secret was not available
	BackendSecretNotAvailable = "BackendSecretNotAvailable"
	// UnableToCheckBackendSecretAvailability indicates that the condition transitioned to this state because operator was unable
	// to check the backend Secret availability
	UnableToCheckBackendSecretAvailability = "UnableToCheckBackendSecretAvailability"

	// ResourceValidationPassed indicates that the condition transitioned to this state because the CRD meets validation criteria
	ResourceValidationPassed = "ResourceValidationPassed"
	// ResourceValidationFailed indicates that the condition transitioned to this state because the CRD does not meet validation criteria
	ResourceValidationFailed = "ResourceValidationFailed"
)
