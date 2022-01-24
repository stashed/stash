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
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              RestoreSessionSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status            RestoreSessionStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type RestoreSessionSpec struct {
	RestoreTargetSpec `json:",inline,omitempty" protobuf:"bytes,1,opt,name=restoreTargetSpec"`
	// Driver indicates the name of the agent to use to restore the target.
	// Supported values are "Restic", "VolumeSnapshotter".
	// Default value is "Restic".
	// +optional
	// +kubebuilder:default=Restic
	Driver Snapshotter `json:"driver,omitempty" protobuf:"bytes,2,opt,name=driver,casttype=Snapshotter"`
	// Repository refer to the Repository crd that hold backend information
	// +optional
	Repository kmapi.ObjectReference `json:"repository,omitempty" protobuf:"bytes,3,opt,name=repository"`
	// Rules specifies different restore options for different hosts
	// +optional
	// Deprecated. Use rules section inside `target`.
	Rules []Rule `json:"rules,omitempty" protobuf:"bytes,4,rep,name=rules"`
}

type RestoreTargetSpec struct {
	// Task specify the Task crd that specifies the steps for recovery process
	// +optional
	Task TaskRef `json:"task,omitempty" protobuf:"bytes,1,opt,name=task"`
	// Target indicates the target where the recovered data will be stored
	// +optional
	Target *RestoreTarget `json:"target,omitempty" protobuf:"bytes,2,opt,name=target"`
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
	// Actions that Stash should take in response to restore sessions.
	// +optional
	Hooks *RestoreHooks `json:"hooks,omitempty" protobuf:"bytes,6,opt,name=hooks"`
}

// Hooks describes actions that Stash should take in response to restore sessions. For the PostRestore
// and PreRestore handlers, restore process blocks until the action is complete,
// unless the container process fails, in which case the handler is aborted.
type RestoreHooks struct {
	// PreRestore is called immediately before a restore session is initiated.
	// +optional
	PreRestore *prober.Handler `json:"preRestore,omitempty" protobuf:"bytes,1,opt,name=preRestore"`

	// PostRestore is called immediately after a restore session is complete.
	// +optional
	PostRestore *prober.Handler `json:"postRestore,omitempty" protobuf:"bytes,2,opt,name=postRestore"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RestoreSessionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []RestoreSession `json:"items,omitempty" protobuf:"bytes,2,rep,name=items"`
}

// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed;Unknown
type RestorePhase string

const (
	RestorePending      RestorePhase = "Pending"
	RestoreRunning      RestorePhase = "Running"
	RestoreSucceeded    RestorePhase = "Succeeded"
	RestoreFailed       RestorePhase = "Failed"
	RestorePhaseUnknown RestorePhase = "Unknown"
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
	Phase RestorePhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=RestorePhase"`
	// TotalHosts specifies total number of hosts that will be restored for this RestoreSession
	// +optional
	TotalHosts *int32 `json:"totalHosts,omitempty" protobuf:"varint,2,opt,name=totalHosts"`
	// SessionDuration specify total time taken to complete current restore session (sum of restore duration of all hosts)
	// +optional
	SessionDuration string `json:"sessionDuration,omitempty" protobuf:"bytes,3,opt,name=sessionDuration"`
	// Stats shows statistics of individual hosts for this restore session
	// +optional
	Stats []HostRestoreStats `json:"stats,omitempty" protobuf:"bytes,4,rep,name=stats"`
	// Conditions shows current restore condition of the RestoreSession.
	// +optional
	Conditions []kmapi.Condition `json:"conditions,omitempty" protobuf:"bytes,5,rep,name=conditions"`
}

type HostRestoreStats struct {
	// Hostname indicate name of the host that has been restored
	// +optional
	Hostname string `json:"hostname,omitempty" protobuf:"bytes,1,opt,name=hostname"`
	// Phase indicates restore phase of this host
	// +optional
	Phase HostRestorePhase `json:"phase,omitempty" protobuf:"bytes,2,opt,name=phase,casttype=HostRestorePhase"`
	// Duration indicates total time taken to complete restore for this hosts
	// +optional
	Duration string `json:"duration,omitempty" protobuf:"bytes,3,opt,name=duration"`
	// Error indicates string value of error in case of restore failure
	// +optional
	Error string `json:"error,omitempty" protobuf:"bytes,4,opt,name=error"`
}
