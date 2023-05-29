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
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BackupSessionSpec   `json:"spec,omitempty"`
	Status            BackupSessionStatus `json:"status,omitempty"`
}

type BackupSessionSpec struct {
	// Invoker refers to the BackupConfiguration or BackupBatch being used to invoke this backup session
	// +optional
	Invoker BackupInvokerRef `json:"invoker,omitempty"`

	// RetryLeft specifies number of retry attempts left for the session.
	// If this set to non-zero, Stash will create a new BackupSession if the current one fails.
	// +optional
	RetryLeft int32 `json:"retryLeft,omitempty"`
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
	Phase BackupSessionPhase `json:"phase,omitempty"`
	// SessionDuration specify total time taken to complete current backup session (sum of backup duration of all targets)
	// +optional
	SessionDuration string `json:"sessionDuration,omitempty"`
	// Targets specify the backup status of individual targets
	// +optional
	Targets []BackupTargetStatus `json:"targets,omitempty"`
	// Conditions shows condition of different operations/steps of the backup process
	// +optional
	Conditions []kmapi.Condition `json:"conditions,omitempty"`
	// SessionDeadline specifies the deadline of backup. BackupSession will be
	// considered Failed if backup does not complete within this deadline
	// +optional
	SessionDeadline *metav1.Time `json:"sessionDeadline,omitempty"`

	// Retried specifies whether this session was retried or not.
	// This field will exist only if the `retryConfig` has been set in the respective backup invoker.
	// +optional
	Retried *bool `json:"retried,omitempty"`

	// NextRetry specifies the time when Stash should retry the current failed backup.
	// This field will exist only if the `retryConfig` has been set in the respective backup invoker.
	// +optional
	NextRetry *metav1.Time `json:"nextRetry,omitempty"`
}

type BackupTargetStatus struct {
	// Ref refers to the backup target
	// +optional
	Ref TargetRef `json:"ref,omitempty"`
	// TotalHosts specifies total number of hosts for this target that will be backed up for a BackupSession
	// +optional
	TotalHosts *int32 `json:"totalHosts,omitempty"`
	// Phase indicates backup phase of this target
	// +optional
	Phase TargetPhase `json:"phase,omitempty"`
	// Stats shows statistics of individual hosts for this backup session
	// +optional
	Stats []HostBackupStats `json:"stats,omitempty"`
	// PreBackupActions specifies a list of actions that the backup process should execute before taking backup
	// +optional
	PreBackupActions []string `json:"preBackupActions,omitempty"`
	// PostBackupActions specifies a list of actions that the backup process should execute after taking backup
	// +optional
	PostBackupActions []string `json:"postBackupActions,omitempty"`
	// Conditions shows condition of different operations/steps of the backup process for this target
	// +optional
	Conditions []kmapi.Condition `json:"conditions,omitempty"`
}

type HostBackupStats struct {
	// Hostname indicate name of the host that has been backed up
	// +optional
	Hostname string `json:"hostname,omitempty"`
	// Phase indicates backup phase of this host
	// +optional
	Phase HostBackupPhase `json:"phase,omitempty"`
	// Snapshots specifies the stats of individual snapshots that has been taken for this host in current backup session
	// +optional
	Snapshots []SnapshotStats `json:"snapshots,omitempty"`
	// Duration indicates total time taken to complete backup for this host
	// +optional
	Duration string `json:"duration,omitempty"`
	// Error indicates string value of error in case of backup failure
	// +optional
	Error string `json:"error,omitempty"`
}

type SnapshotStats struct {
	// Name indicates the name of the backup snapshot created for this host
	Name string `json:"name,omitempty"`
	// Path indicates the directory that has been backed up in this snapshot
	Path string `json:"path,omitempty"`
	// TotalSize indicates the size of data to backup in target directory
	TotalSize string `json:"totalSize,omitempty"`
	// Uploaded indicates size of data uploaded to backend for this snapshot
	Uploaded string `json:"uploaded,omitempty"`
	// ProcessingTime indicates time taken to process the target data
	ProcessingTime string `json:"processingTime,omitempty"`
	// FileStats shows statistics of files of this snapshot
	FileStats FileStats `json:"fileStats,omitempty"`
}

type FileStats struct {
	// TotalFiles shows total number of files that has been backed up
	TotalFiles *int64 `json:"totalFiles,omitempty"`
	// NewFiles shows total number of new files that has been created since last backup
	NewFiles *int64 `json:"newFiles,omitempty"`
	// ModifiedFiles shows total number of files that has been modified since last backup
	ModifiedFiles *int64 `json:"modifiedFiles,omitempty"`
	// UnmodifiedFiles shows total number of files that has not been changed since last backup
	UnmodifiedFiles *int64 `json:"unmodifiedFiles,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupSessionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupSession `json:"items,omitempty"`
}

// =============================== Pre-backup Actions ============================
const (
	InitializeBackendRepository = "InitializeBackendRepository"
)

// ============================== Post-backup Actions ==========================
const (
	ApplyRetentionPolicy      = "ApplyRetentionPolicy"
	VerifyRepositoryIntegrity = "VerifyRepositoryIntegrity"
	SendRepositoryMetrics     = "SendRepositoryMetrics"
)

// ============================ Condition Types ========================
const (
	// RetentionPolicyApplied indicates that whether the retention policies were applied or not
	RetentionPolicyApplied = "RetentionPolicyApplied"

	// BackendRepositoryInitialized indicates that whether backend repository was initialized or not
	BackendRepositoryInitialized = "BackendRepositoryInitialized"

	// RepositoryIntegrityVerified indicates whether the repository integrity check succeeded or not
	RepositoryIntegrityVerified = "RepositoryIntegrityVerified"

	// RepositoryMetricsPushed indicates whether the Repository metrics for this backup session were pushed or not
	RepositoryMetricsPushed = "RepositoryMetricsPushed"

	// BackupSkipped indicates that the current session was skipped
	BackupSkipped = "BackupSkipped"

	// BackupHistoryCleaned indicates whether the backup history was cleaned or not according to backupHistoryLimit
	BackupHistoryCleaned = "BackupHistoryCleaned"

	// BackupExecutorEnsured indicates whether the backup executor entity was created or not
	BackupExecutorEnsured = "BackupExecutorEnsured"

	// PreBackupHookExecutionSucceeded indicates whether the preBackup hook was executed successfully or not
	PreBackupHookExecutionSucceeded = "PreBackupHookExecutionSucceeded"

	// PostBackupHookExecutionSucceeded indicates whether the postBackup hook was executed successfully or not
	PostBackupHookExecutionSucceeded = "PostBackupHookExecutionSucceeded"

	// DeadlineExceeded  indicates whether the session deadline was exceeded or not
	DeadlineExceeded = "DeadlineExceeded"
)

// =========================== Condition Reasons =======================
const (
	// SuccessfullyAppliedRetentionPolicy indicates that the condition transitioned to this state because the retention policies was applied successfully
	SuccessfullyAppliedRetentionPolicy = "SuccessfullyAppliedRetentionPolicy"
	// FailedToApplyRetentionPolicy indicates that the condition transitioned to this state because the Stash was unable to apply the retention policies
	FailedToApplyRetentionPolicy = "FailedToApplyRetentionPolicy"

	// BackendRepositoryFound indicates that the condition transitioned to this state because the restic repository was found in the backend
	BackendRepositoryFound = "BackendRepositoryFound"
	// FailedToInitializeBackendRepository indicates that the condition transitioned to this state because the Stash was unable to initialize a repository in the backend
	FailedToInitializeBackendRepository = "FailedToInitializeBackendRepository"

	// SuccessfullyVerifiedRepositoryIntegrity indicates that the condition transitioned to this state because the repository has passed the integrity check
	SuccessfullyVerifiedRepositoryIntegrity = "SuccessfullyVerifiedRepositoryIntegrity"
	// FailedToVerifyRepositoryIntegrity indicates that the condition transitioned to this state because the repository has failed the integrity check
	FailedToVerifyRepositoryIntegrity = "FailedToVerifyRepositoryIntegrity"

	// SuccessfullyPushedRepositoryMetrics indicates that the condition transitioned to this state because the repository metrics was successfully pushed to the pushgateway
	SuccessfullyPushedRepositoryMetrics = "SuccessfullyPushedRepositoryMetrics"
	// FailedToPushRepositoryMetrics indicates that the condition transitioned to this state because the Stash was unable to push the repository metrics to the pushgateway
	FailedToPushRepositoryMetrics = "FailedToPushRepositoryMetrics"

	// SkippedTakingNewBackup indicates that the backup was skipped because another backup was running or backup invoker is not ready state.
	SkippedTakingNewBackup = "SkippedTakingNewBackup"

	SuccessfullyCleanedBackupHistory = "SuccessfullyCleanedBackupHistory"
	FailedToCleanBackupHistory       = "FailedToCleanBackupHistory"

	SuccessfullyEnsuredBackupExecutor = "SuccessfullyEnsuredBackupExecutor"
	FailedToEnsureBackupExecutor      = "FailedToEnsureBackupExecutor"

	SuccessfullyExecutedPreBackupHook = "SuccessfullyExecutedPreBackupHook"
	FailedToExecutePreBackupHook      = "FailedToExecutePreBackupHook"

	SuccessfullyExecutedPostBackupHook = "SuccessfullyExecutedPostBackupHook"
	FailedToExecutePostBackupHook      = "FailedToExecutePostBackupHook"

	FailedToCompleteWithinDeadline = "FailedToCompleteWithinDeadline"
)
