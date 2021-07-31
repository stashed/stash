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
	ResourceKindBackupSession     = "BackupSession"
	ResourceSingularBackupSession = "backupsession"
	ResourcePluralBackupSession   = "backupsessions"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=backupsessions,singular=backupsession,categories={stash,appscode,all}
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Invoker-Type",type="string",JSONPath=".spec.invoker.kind"
// +kubebuilder:printcolumn:name="Invoker-Name",type="string",JSONPath=".spec.invoker.name"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Duration",type="string",JSONPath=".status.sessionDuration"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type BackupSession struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              BackupSessionSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status            BackupSessionStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type BackupSessionSpec struct {
	// Invoker refers to the BackupConfiguration or BackupBatch being used to invoke this backup session
	// +optional
	Invoker BackupInvokerRef `json:"invoker,omitempty" protobuf:"bytes,1,opt,name=invoker"`
}

// +kubebuilder:validation:Enum=Pending;Skipped;Running;Succeeded;Failed;Unknown
type BackupSessionPhase string

const (
	BackupSessionPending   BackupSessionPhase = "Pending"
	BackupSessionSkipped   BackupSessionPhase = "Skipped"
	BackupSessionRunning   BackupSessionPhase = "Running"
	BackupSessionSucceeded BackupSessionPhase = "Succeeded"
	BackupSessionFailed    BackupSessionPhase = "Failed"
	BackupSessionUnknown   BackupSessionPhase = "Unknown"
)

// +kubebuilder:validation:Enum=Succeeded;Failed
type HostBackupPhase string

const (
	HostBackupSucceeded HostBackupPhase = "Succeeded"
	HostBackupFailed    HostBackupPhase = "Failed"
)

// +kubebuilder:validation:Enum=Pending;Succeeded;Running;Failed
type TargetPhase string

const (
	TargetBackupPending   TargetPhase = "Pending"
	TargetBackupSucceeded TargetPhase = "Succeeded"
	TargetBackupRunning   TargetPhase = "Running"
	TargetBackupFailed    TargetPhase = "Failed"
)

type BackupSessionStatus struct {
	// Phase indicates the overall phase of the backup process for this BackupSession. Phase will be "Succeeded" only if
	// phase of all hosts are "Succeeded". If any of the host fail to complete backup, Phase will be "Failed".
	// +optional
	Phase BackupSessionPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=BackupSessionPhase"`
	// SessionDuration specify total time taken to complete current backup session (sum of backup duration of all targets)
	// +optional
	SessionDuration string `json:"sessionDuration,omitempty" protobuf:"bytes,2,opt,name=sessionDuration"`
	// Targets specify the backup status of individual targets
	// +optional
	Targets []BackupTargetStatus `json:"targets,omitempty" protobuf:"bytes,3,rep,name=targets"`
	// Conditions shows condition of different operations/steps of the backup process
	// +optional
	Conditions []kmapi.Condition `json:"conditions,omitempty" protobuf:"bytes,4,rep,name=conditions"`
}

type BackupTargetStatus struct {
	// Ref refers to the backup target
	// +optional
	Ref TargetRef `json:"ref,omitempty" protobuf:"bytes,1,opt,name=ref"`
	// TotalHosts specifies total number of hosts for this target that will be backed up for a BackupSession
	// +optional
	TotalHosts *int32 `json:"totalHosts,omitempty" protobuf:"varint,2,opt,name=totalHosts"`
	// Phase indicates backup phase of this target
	// +optional
	Phase TargetPhase `json:"phase,omitempty" protobuf:"bytes,3,opt,name=phase"`
	// Stats shows statistics of individual hosts for this backup session
	// +optional
	Stats []HostBackupStats `json:"stats,omitempty" protobuf:"bytes,4,rep,name=stats"`
	// PreBackupActions specifies a list of actions that the backup process should execute before taking backup
	// +optional
	PreBackupActions []string `json:"preBackupActions,omitempty" protobuf:"bytes,5,rep,name=preBackupActions"`
	// PostBackupActions specifies a list of actions that the backup process should execute after taking backup
	// +optional
	PostBackupActions []string `json:"postBackupActions,omitempty" protobuf:"bytes,6,rep,name=postBackupActions"`
}

type HostBackupStats struct {
	// Hostname indicate name of the host that has been backed up
	// +optional
	Hostname string `json:"hostname,omitempty" protobuf:"bytes,1,opt,name=hostname"`
	// Phase indicates backup phase of this host
	// +optional
	Phase HostBackupPhase `json:"phase,omitempty" protobuf:"bytes,2,opt,name=phase,casttype=HostBackupPhase"`
	// Snapshots specifies the stats of individual snapshots that has been taken for this host in current backup session
	// +optional
	Snapshots []SnapshotStats `json:"snapshots,omitempty" protobuf:"bytes,3,rep,name=snapshots"`
	// Duration indicates total time taken to complete backup for this hosts
	// +optional
	Duration string `json:"duration,omitempty" protobuf:"bytes,4,opt,name=duration"`
	// Error indicates string value of error in case of backup failure
	// +optional
	Error string `json:"error,omitempty" protobuf:"bytes,5,opt,name=error"`
}

type SnapshotStats struct {
	// Name indicates the name of the backup snapshot created for this host
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// Path indicates the directory that has been backed up in this snapshot
	Path string `json:"path,omitempty" protobuf:"bytes,2,opt,name=path"`
	// TotalSize indicates the size of data to backup in target directory
	TotalSize string `json:"totalSize,omitempty" protobuf:"bytes,7,opt,name=totalSize"`
	// Uploaded indicates size of data uploaded to backend for this snapshot
	Uploaded string `json:"uploaded,omitempty" protobuf:"bytes,4,opt,name=uploaded"`
	// ProcessingTime indicates time taken to process the target data
	ProcessingTime string `json:"processingTime,omitempty" protobuf:"bytes,5,opt,name=processingTime"`
	// FileStats shows statistics of files of this snapshot
	FileStats FileStats `json:"fileStats,omitempty" protobuf:"bytes,6,opt,name=fileStats"`
}

type FileStats struct {
	// TotalFiles shows total number of files that has been backed up
	TotalFiles *int64 `json:"totalFiles,omitempty" protobuf:"varint,1,opt,name=totalFiles"`
	// NewFiles shows total number of new files that has been created since last backup
	NewFiles *int64 `json:"newFiles,omitempty" protobuf:"varint,2,opt,name=newFiles"`
	// ModifiedFiles shows total number of files that has been modified since last backup
	ModifiedFiles *int64 `json:"modifiedFiles,omitempty" protobuf:"varint,3,opt,name=modifiedFiles"`
	// UnmodifiedFiles shows total number of files that has not been changed since last backup
	UnmodifiedFiles *int64 `json:"unmodifiedFiles,omitempty" protobuf:"varint,4,opt,name=unmodifiedFiles"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupSessionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []BackupSession `json:"items,omitempty" protobuf:"bytes,2,rep,name=items"`
}
