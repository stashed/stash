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
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type BackupConfiguration struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              BackupConfigurationSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status            BackupConfigurationStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type BackupConfigurationTemplateSpec struct {
	// Task specify the Task crd that specifies the steps to take backup
	// +optional
	Task TaskRef `json:"task,omitempty" protobuf:"bytes,1,opt,name=task"`
	// Target specify the backup target
	// +optional
	Target *BackupTarget `json:"target,omitempty" protobuf:"bytes,2,opt,name=target"`
	// RuntimeSettings allow to specify Resources, NodeSelector, Affinity, Toleration, ReadinessProbe etc.
	// +optional
	RuntimeSettings ofst.RuntimeSettings `json:"runtimeSettings,omitempty" protobuf:"bytes,3,opt,name=runtimeSettings"`
	// Temp directory configuration for functions/sidecar
	// An `EmptyDir` will always be mounted at /tmp with this settings
	// +optional
	TempDir EmptyDirSettings `json:"tempDir,omitempty" protobuf:"bytes,4,opt,name=tempDir"`
	// InterimVolumeTemplate specifies a template for a volume to hold targeted data temporarily
	// before uploading to backend or inserting into target. It is only usable for job model.
	// Don't specify it in sidecar model.
	// +optional
	InterimVolumeTemplate *ofst.PersistentVolumeClaim `json:"interimVolumeTemplate,omitempty" protobuf:"bytes,5,opt,name=interimVolumeTemplate"`
	// Actions that Stash should take in response to backup sessions.
	// +optional
	Hooks *BackupHooks `json:"hooks,omitempty" protobuf:"bytes,6,opt,name=hooks"`
}

type BackupConfigurationSpec struct {
	BackupConfigurationTemplateSpec `json:",inline,omitempty" protobuf:"bytes,1,opt,name=backupConfigurationTemplateSpec"`
	// Schedule specifies the schedule for invoking backup sessions
	// +optional
	Schedule string `json:"schedule,omitempty" protobuf:"bytes,2,opt,name=schedule"`
	// Driver indicates the name of the agent to use to backup the target.
	// Supported values are "Restic", "VolumeSnapshotter".
	// Default value is "Restic".
	// +optional
	// +kubebuilder:default=Restic
	Driver Snapshotter `json:"driver,omitempty" protobuf:"bytes,3,opt,name=driver,casttype=Snapshotter"`
	// Repository refer to the Repository crd that holds backend information
	// +optional
	Repository kmapi.ObjectReference `json:"repository,omitempty" protobuf:"bytes,4,opt,name=repository"`
	// RetentionPolicy indicates the policy to follow to clean old backup snapshots
	RetentionPolicy v1alpha1.RetentionPolicy `json:"retentionPolicy" protobuf:"bytes,5,opt,name=retentionPolicy"`
	// Indicates that the BackupConfiguration is paused from taking backup. Default value is 'false'
	// +optional
	Paused bool `json:"paused,omitempty" protobuf:"varint,6,opt,name=paused"`
	// BackupHistoryLimit specifies the number of BackupSession and it's associate resources to keep.
	// This is helpful for debugging purpose.
	// Default: 1
	// +optional
	BackupHistoryLimit *int32 `json:"backupHistoryLimit,omitempty" protobuf:"varint,7,opt,name=backupHistoryLimit"`
}

// Hooks describes actions that Stash should take in response to backup sessions. For the PostBackup
// and PreBackup handlers, backup process blocks until the action is complete,
// unless the container process fails, in which case the handler is aborted.
type BackupHooks struct {
	// PreBackup is called immediately before a backup session is initiated.
	// +optional
	PreBackup *prober.Handler `json:"preBackup,omitempty" protobuf:"bytes,1,opt,name=preBackup"`

	// PostBackup is called immediately after a backup session is complete.
	// +optional
	PostBackup *prober.Handler `json:"postBackup,omitempty" protobuf:"bytes,2,opt,name=postBackup"`
}

type EmptyDirSettings struct {
	Medium    core.StorageMedium `json:"medium,omitempty" protobuf:"bytes,1,opt,name=medium,casttype=k8s.io/api/core/v1.StorageMedium"`
	SizeLimit *resource.Quantity `json:"sizeLimit,omitempty" protobuf:"bytes,2,opt,name=sizeLimit"`
	// More info: https://github.com/restic/restic/blob/master/doc/manual_rest.rst#caching
	DisableCaching bool `json:"disableCaching,omitempty" protobuf:"varint,3,opt,name=disableCaching"`
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
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,1,opt,name=observedGeneration"`
	// Conditions shows current backup setup condition of the BackupConfiguration.
	// +optional
	Conditions []kmapi.Condition `json:"conditions,omitempty" protobuf:"bytes,2,rep,name=conditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []BackupConfiguration `json:"items,omitempty" protobuf:"bytes,2,rep,name=items"`
}
