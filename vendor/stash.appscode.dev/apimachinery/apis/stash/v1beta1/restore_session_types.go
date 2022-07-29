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
	ofst "kmodules.xyz/offshoot-api/api/v1"
	prober "kmodules.xyz/prober/api/v1"
)

const (
	ResourceKindRestoreSession     = "RestoreSession"
	ResourcePluralRestoreSession   = "restoresessions"
	ResourceSingularRestoreSession = "restoresession"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=restoresessions,singular=restoresession,shortName=restore,categories={stash,appscode,all}
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Repository",type="string",JSONPath=".spec.repository.name"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Duration",type="string",JSONPath=".status.sessionDuration"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type RestoreSession struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RestoreSessionSpec   `json:"spec,omitempty"`
	Status            RestoreSessionStatus `json:"status,omitempty"`
}

type RestoreSessionSpec struct {
	RestoreTargetSpec `json:",inline,omitempty"`
	// Driver indicates the name of the agent to use to restore the target.
	// Supported values are "Restic", "VolumeSnapshotter".
	// Default value is "Restic".
	// +optional
	// +kubebuilder:default=Restic
	Driver Snapshotter `json:"driver,omitempty"`
	// Repository refer to the Repository crd that hold backend information
	// +optional
	Repository kmapi.ObjectReference `json:"repository,omitempty"`
	// Rules specifies different restore options for different hosts
	// +optional
	// Deprecated. Use rules section inside `target`.
	Rules []Rule `json:"rules,omitempty"`
	// TimeOut specifies the maximum duration of restore. RestoreSession will be considered Failed
	// if restore does not complete within this time limit. By default, Stash don't set any timeout for restore.
	// +optional
	TimeOut string `json:"timeOut,omitempty"`
}

type RestoreTargetSpec struct {
	// Task specify the Task crd that specifies the steps for recovery process
	// +optional
	Task TaskRef `json:"task,omitempty"`
	// Target indicates the target where the recovered data will be stored
	// +optional
	Target *RestoreTarget `json:"target,omitempty"`
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
	// Actions that Stash should take in response to restore sessions.
	// +optional
	Hooks *RestoreHooks `json:"hooks,omitempty"`
}

// Hooks describes actions that Stash should take in response to restore sessions. For the PostRestore
// and PreRestore handlers, restore process blocks until the action is complete,
// unless the container process fails, in which case the handler is aborted.
type RestoreHooks struct {
	// PreRestore is called immediately before a restore session is initiated.
	// +optional
	PreRestore *prober.Handler `json:"preRestore,omitempty"`

	// PostRestore is called according to executionPolicy after a restore session is complete.
	// +optional
	PostRestore *PostRestoreHook `json:"postRestore,omitempty"`
}

type PostRestoreHook struct {
	*prober.Handler `json:",inline"`
	// ExecutionPolicy specifies when to execute a hook.
	// Supported values are "Always", "OnFailure", "OnSuccess".
	// Default value: "Always".
	// +optional
	// +kubebuilder:default=Always
	// +kubebuilder:validation:Enum=Always;OnFailure;OnSuccess
	ExecutionPolicy HookExecutionPolicy `json:"executionPolicy,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RestoreSessionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RestoreSession `json:"items,omitempty"`
}

// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed;Unknown;Invalid
type RestorePhase string

const (
	RestorePending      RestorePhase = "Pending"
	RestoreRunning      RestorePhase = "Running"
	RestoreSucceeded    RestorePhase = "Succeeded"
	RestoreFailed       RestorePhase = "Failed"
	RestorePhaseUnknown RestorePhase = "Unknown"
	RestorePhaseInvalid RestorePhase = "Invalid"
)

// +kubebuilder:validation:Enum=Succeeded;Failed;Running;Unknown
type HostRestorePhase string

const (
	HostRestoreSucceeded HostRestorePhase = "Succeeded"
	HostRestoreFailed    HostRestorePhase = "Failed"
	HostRestoreRunning   HostRestorePhase = "Running"
	HostRestoreUnknown   HostRestorePhase = "Unknown"
)

type RestoreSessionStatus struct {
	// Phase indicates the overall phase of the restore process for this RestoreSession. Phase will be "Succeeded" only if
	// phase of all hosts are "Succeeded". If any of the host fail to complete restore, Phase will be "Failed".
	// +optional
	Phase RestorePhase `json:"phase,omitempty"`
	// TotalHosts specifies total number of hosts that will be restored for this RestoreSession
	// +optional
	TotalHosts *int32 `json:"totalHosts,omitempty"`
	// SessionDuration specify total time taken to complete current restore session (sum of restore duration of all hosts)
	// +optional
	SessionDuration string `json:"sessionDuration,omitempty"`
	// Stats shows statistics of individual hosts for this restore session
	// +optional
	Stats []HostRestoreStats `json:"stats,omitempty"`
	// Conditions shows current restore condition of the RestoreSession.
	// +optional
	Conditions []kmapi.Condition `json:"conditions,omitempty"`
	// SessionDeadline specifies the deadline of restore process. RestoreSession will be
	// considered Failed if restore does not complete within this deadline
	// +optional
	SessionDeadline metav1.Time `json:"sessionDeadline,omitempty"`
}

type HostRestoreStats struct {
	// Hostname indicate name of the host that has been restored
	// +optional
	Hostname string `json:"hostname,omitempty"`
	// Phase indicates restore phase of this host
	// +optional
	Phase HostRestorePhase `json:"phase,omitempty"`
	// Duration indicates total time taken to complete restore for this hosts
	// +optional
	Duration string `json:"duration,omitempty"`
	// Error indicates string value of error in case of restore failure
	// +optional
	Error string `json:"error,omitempty"`
}

// ========================= Condition Types ===================
const (
	// RestoreTargetFound indicates whether the restore target was found
	RestoreTargetFound = "RestoreTargetFound"

	// StashInitContainerInjected indicates whether stash init-container was injected into the targeted workload
	// This condition is applicable only for sidecar model
	StashInitContainerInjected = "StashInitContainerInjected"

	// RestoreJobCreated indicates whether the restore job was created
	RestoreJobCreated = "RestoreJobCreated"

	// RestoreCompleted condition indicates whether the restore process has been completed or not.
	// This condition is particularly helpful when the restore addon require some additional operations to perform
	// before marking the RestoreSession Succeeded/Failed.
	RestoreCompleted = "RestoreCompleted"

	// RestoreExecutorEnsured condition indicates whether the restore job / init-container was ensured or not.
	RestoreExecutorEnsured = "RestoreExecutorEnsured"

	// MetricsPushed whether the metrics for this backup session were pushed or not
	MetricsPushed = "MetricsPushed"

	// PreRestoreHookExecutionSucceeded indicates whether the preRestore hook was executed successfully or not
	PreRestoreHookExecutionSucceeded = "PreRestoreHookExecutionSucceeded"

	// PostRestoreHookExecutionSucceeded indicates whether the postRestore hook was executed successfully or not
	PostRestoreHookExecutionSucceeded = "PostRestoreHookExecutionSucceeded"
)

// ======================== Condition Reasons ===================
const (
	// InitContainerInjectionSucceeded indicates that the condition transitioned to this state because stash init-container
	// was injected successfully into the targeted workload
	InitContainerInjectionSucceeded = "InitContainerInjectionSucceeded"
	// InitContainerInjectionFailed indicates that the condition transitioned to this state because operator was unable
	// to inject stash init-container into the targeted workload
	InitContainerInjectionFailed = "InitContainerInjectionFailed"

	// RestoreJobCreationSucceeded indicates that the condition transitioned to this state because restore job was created successfully
	RestoreJobCreationSucceeded = "RestoreJobCreationSucceeded"
	// RestoreJobCreationFailed indicates that the condition transitioned to this state because operator was unable to create restore job
	RestoreJobCreationFailed = "RestoreJobCreationFailed"

	// SuccessfullyPushedMetrics indicates that the condition transitioned to this state because the metrics was successfully pushed to the pushgateway
	SuccessfullyPushedMetrics = "SuccessfullyPushedMetrics"
	// FailedToPushMetrics indicates that the condition transitioned to this state because the Stash was unable to push the metrics to the pushgateway
	FailedToPushMetrics = "FailedToPushMetrics"

	SuccessfullyEnsuredRestoreExecutor = "SuccessfullyEnsuredRestoreExecutor"
	FailedToEnsureRestoreExecutor      = "FailedToEnsureRestoreExecutor"

	SuccessfullyExecutedPreRestoreHook = "SuccessfullyExecutedPreRestoreHook"
	FailedToExecutePreRestoreHook      = "FailedToExecutePreRestoreHook"

	SuccessfullyExecutedPostRestoreHook = "SuccessfullyExecutedPostRestoreHook"
	FailedToExecutePostRestoreHook      = "FailedToExecutePostRestoreHook"

	PostRestoreTasksExecuted    = "PostRestoreTasksExecuted"
	PostRestoreTasksNotExecuted = "PostRestoreTasksNotExecuted"
)
