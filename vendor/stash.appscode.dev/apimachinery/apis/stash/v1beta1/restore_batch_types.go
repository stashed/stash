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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
)

const (
	ResourceKindRestoreBatch     = "RestoreBatch"
	ResourceSingularRestoreBatch = "restorebatch"
	ResourcePluralRestoreBatch   = "restorebatches"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=restorebatches,singular=restorebatch,categories={stash,appscode,all}
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Repository",type="string",JSONPath=".spec.repository.name"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type RestoreBatch struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RestoreBatchSpec   `json:"spec,omitempty"`
	Status            RestoreBatchStatus `json:"status,omitempty"`
}

type RestoreBatchSpec struct {
	// Driver indicates the name of the agent to use to restore the target.
	// Supported values are "Restic", "VolumeSnapshotter".
	// Default value is "Restic".
	// +optional
	// +kubebuilder:default=Restic
	Driver Snapshotter `json:"driver,omitempty"`
	// Repository refer to the Repository crd that holds backend information
	// +optional
	Repository kmapi.ObjectReference `json:"repository,omitempty"`
	// Members is a list of restore targets and their configuration that are part of this batch
	// +optional
	Members []RestoreTargetSpec `json:"members,omitempty"`
	// ExecutionOrder indicate whether to restore the members in the sequential order as they appear in the members list.
	// The default value is "Parallel" which means the members will be restored in parallel.
	// +kubebuilder:default=Parallel
	// +optional
	ExecutionOrder ExecutionOrder `json:"executionOrder,omitempty"`
	// Hooks specifies the actions that Stash should take before or after restore.
	// Cannot be updated.
	// +optional
	Hooks *RestoreHooks `json:"hooks,omitempty"`
}

type RestoreBatchStatus struct {
	// Phase indicates the overall phase of the restore process for this RestoreBatch. Phase will be "Succeeded" only if
	// phase of all members are "Succeeded". If the restore process fail for any of the members, Phase will be "Failed".
	// +optional
	Phase RestorePhase `json:"phase,omitempty"`
	// SessionDuration specify total time taken to complete restore of all the members.
	// +optional
	SessionDuration string `json:"sessionDuration,omitempty"`
	// Conditions shows the condition of different steps for the RestoreBatch.
	// +optional
	Conditions []kmapi.Condition `json:"conditions,omitempty"`
	// Members shows the restore status for the members of the RestoreBatch.
	// +optional
	Members []RestoreMemberStatus `json:"members,omitempty"`
}

// +kubebuilder:validation:Enum=Pending;Succeeded;Running;Failed
type RestoreTargetPhase string

const (
	TargetRestorePending      RestoreTargetPhase = "Pending"
	TargetRestoreRunning      RestoreTargetPhase = "Running"
	TargetRestoreSucceeded    RestoreTargetPhase = "Succeeded"
	TargetRestoreFailed       RestoreTargetPhase = "Failed"
	TargetRestorePhaseUnknown RestoreTargetPhase = "Unknown"
)

type RestoreMemberStatus struct {
	// Ref is the reference to the respective target whose status is shown here.
	Ref TargetRef `json:"ref"`
	// Conditions shows the condition of different steps to restore this member.
	// +optional
	Conditions []kmapi.Condition `json:"conditions,omitempty"`
	// TotalHosts specifies total number of hosts that will be restored for this member.
	// +optional
	TotalHosts *int32 `json:"totalHosts,omitempty"`
	// Phase indicates restore phase of this member
	// +optional
	Phase RestoreTargetPhase `json:"phase,omitempty"`
	// Stats shows restore statistics of individual hosts for this member
	// +optional
	Stats []HostRestoreStats `json:"stats,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RestoreBatchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RestoreBatch `json:"items,omitempty"`
}
