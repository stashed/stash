package v1beta1

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
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
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type RestoreSession struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RestoreSessionSpec   `json:"spec,omitempty"`
	Status            RestoreSessionStatus `json:"status,omitempty"`
}

type RestoreSessionSpec struct {
	// Driver indicates the name of the agent to use to restore the target.
	// Supported values are "Restic", "VolumeSnapshotter".
	// Default value is "Restic".
	// +optional
	Driver Snapshotter `json:"driver,omitempty"`
	// Repository refer to the Repository crd that hold backend information
	// +optional
	Repository core.LocalObjectReference `json:"repository,omitempty"`
	// Task specify the Task crd that specifies the steps for recovery process
	// +optional
	Task TaskRef `json:"task,omitempty"`
	// Target indicates the target where the recovered data will be stored
	// +optional
	Target *RestoreTarget `json:"target,omitempty"`
	// Rules specifies different restore options for different hosts
	// +optional
	Rules []Rule `json:"rules,omitempty"`
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
	InterimVolumeTemplate *core.PersistentVolumeClaim `json:"interimVolumeTemplate,omitempty"`
}

type Rule struct {
	// Subjects specifies the list of hosts that are subject to this rule
	// +optional
	TargetHosts []string `json:"targetHosts,omitempty"`
	// SourceHost specifies the name of the host whose backed up state we are trying to restore
	// By default, it will indicate the workload itself
	// +optional
	SourceHost string `json:"sourceHost,omitempty"`
	// Snapshots specifies the list of snapshots that will be restored for the host under this rule.
	// Don't specify if you have specified paths field.
	// +optional
	Snapshots []string `json:"snapshots,omitempty"`
	// Paths specifies the paths to be restored for the hosts under this rule.
	// Don't specify if you have specified snapshots field.
	// +optional
	Paths []string `json:"paths,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RestoreSessionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RestoreSession `json:"items,omitempty"`
}

type RestoreSessionPhase string

const (
	RestoreSessionPending   RestoreSessionPhase = "Pending"
	RestoreSessionRunning   RestoreSessionPhase = "Running"
	RestoreSessionSucceeded RestoreSessionPhase = "Succeeded"
	RestoreSessionFailed    RestoreSessionPhase = "Failed"
	RestoreSessionUnknown   RestoreSessionPhase = "Unknown"
)

type HostRestorePhase string

const (
	HostRestoreSucceeded HostRestorePhase = "Succeeded"
	HostRestoreFailed    HostRestorePhase = "Failed"
	HostRestoreUnknown   HostRestorePhase = "Unknown"
)

type RestoreSessionStatus struct {
	// Phase indicates the overall phase of the restore process for this RestoreSession. Phase will be "Succeeded" only if
	// phase of all hosts are "Succeeded". If any of the host fail to complete restore, Phase will be "Failed".
	// +optional
	Phase RestoreSessionPhase `json:"phase,omitempty"`
	// TotalHosts specifies total number of hosts that will be restored for this RestoreSession
	// +optional
	TotalHosts *int32 `json:"totalHosts,omitempty"`
	// SessionDuration specify total time taken to complete current restore session (sum of restore duration of all hosts)
	// +optional
	SessionDuration string `json:"sessionDuration,omitempty"`
	// Stats shows statistics of individual hosts for this restore session
	// +optional
	Stats []HostRestoreStats `json:"stats,omitempty"`
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
