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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

const (
	ResourceKindBackupBatch     = "BackupBatch"
	ResourceSingularBackupBatch = "backupbatch"
	ResourcePluralBackupBatch   = "backupbatches"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=backupbatches,singular=backupbatch,categories={stash,appscode,all}
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=".spec.schedule"
// +kubebuilder:printcolumn:name="Paused",type="boolean",JSONPath=".spec.paused"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type BackupBatch struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              BackupBatchSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status            BackupBatchStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type BackupBatchSpec struct {
	// members is a list of backup configurations that are part of this batch
	// +optional
	Members []BackupConfigurationTemplateSpec `json:"members,omitempty" protobuf:"bytes,1,rep,name=members"`
	// Schedule specifies the schedule for invoking backup sessions
	// +optional
	Schedule string `json:"schedule,omitempty" protobuf:"bytes,2,opt,name=schedule"`
	// RuntimeSettings allow to specify Resources, NodeSelector, Affinity, Toleration, ReadinessProbe etc,
	// and used to create service account for CronJob.
	// +optional
	RuntimeSettings ofst.RuntimeSettings `json:"runtimeSettings,omitempty" protobuf:"bytes,3,opt,name=runtimeSettings"`
	// Driver indicates the name of the agent to use to backup the target.
	// Supported values are "Restic", "VolumeSnapshotter".
	// Default value is "Restic".
	// +optional
	// +kubebuilder:default=Restic
	Driver Snapshotter `json:"driver,omitempty" protobuf:"bytes,4,opt,name=driver,casttype=Snapshotter"`
	// Repository refer to the Repository crd that holds backend information
	// +optional
	Repository kmapi.ObjectReference `json:"repository,omitempty" protobuf:"bytes,5,opt,name=repository"`
	// RetentionPolicy indicates the policy to follow to clean old backup snapshots
	RetentionPolicy v1alpha1.RetentionPolicy `json:"retentionPolicy" protobuf:"bytes,6,opt,name=retentionPolicy"`
	// Indicates that the BackupConfiguration is paused from taking backup. Default value is 'false'
	// +optional
	Paused bool `json:"paused,omitempty" protobuf:"varint,7,opt,name=paused"`
	// BackupHistoryLimit specifies the number of BackupSession and it's associate resources to keep.
	// This is helpful for debugging purpose.
	// Default: 1
	// +optional
	BackupHistoryLimit *int32 `json:"backupHistoryLimit,omitempty" protobuf:"varint,8,opt,name=backupHistoryLimit"`
	// Actions that Stash should take in response to backup sessions.
	// Cannot be updated.
	// +optional
	Hooks *BackupHooks `json:"hooks,omitempty" protobuf:"bytes,9,opt,name=hooks"`
	// ExecutionOrder indicate whether to backup the members in the sequential order as they appear in the members list.
	// The default value is "Parallel" which means the members will be backed up in parallel.
	// +kubebuilder:default=Parallel
	// +optional
	ExecutionOrder ExecutionOrder `json:"executionOrder,omitempty" protobuf:"bytes,10,opt,name=executionOrder"`
}

type BackupBatchStatus struct {
	// ObservedGeneration is the most recent generation observed for this BackupBatch. It corresponds to the
	// BackupBatch's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,1,opt,name=observedGeneration"`
	// Conditions shows current backup setup condition of the BackupBatch.
	// +optional
	Conditions []kmapi.Condition `json:"conditions,omitempty" protobuf:"bytes,2,rep,name=conditions"`
	// MemberConditions shows current backup setup condition of the members of the BackupBatch.
	// +optional
	MemberConditions []MemberConditions `json:"memberConditions,omitempty" protobuf:"bytes,3,rep,name=memberConditions"`
}

type MemberConditions struct {
	// Target is the reference to the respective target whose condition is shown here.
	Target TargetRef `json:"target" protobuf:"bytes,1,opt,name=target"`
	// Conditions shows current backup setup condition of this member.
	// +optional
	Conditions []kmapi.Condition `json:"conditions,omitempty" protobuf:"bytes,2,rep,name=conditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupBatchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []BackupBatch `json:"items,omitempty" protobuf:"bytes,2,rep,name=items"`
}
