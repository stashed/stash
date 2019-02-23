package v1beta1

import (
	"github.com/appscode/go/encoding/json/types"
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

type RestoreSession struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RestoreSessionSpec   `json:"spec,omitempty"`
	Status            RestoreSessionStatus `json:"status,omitempty"`
}

type RestoreSessionSpec struct {
	// Repository refer to the Repository crd that hold backend information
	Repository core.LocalObjectReference `json:"repository,omitempty"`
	// Task specify the Task crd that specifies the steps for recovery process
	// +optional
	Task TaskRef `json:"task,omitempty"`
	// Target indicates the target where the recovered data will be stored
	// +optional
	Target *Target `json:"target,omitempty"`
	// Rules specifies different restore options for different hosts
	// +optional
	Rules []Rule `json:"rules,omitempty"`
	// RuntimeSettings allow to specify Resources, NodeSelector, Affinity, Toleration, ReadinessProbe etc.
	//+optional
	RuntimeSettings ofst.RuntimeSettings `json:"runtimeSettings,omitempty"`
}

type Rule struct {
	// Hosts specifies the list of hosts that are subject to this rule
	// +optional
	Hosts []string `json:"hosts,omitempty"`
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

type RestorePhase string

const (
	RestorePending   RestorePhase = "Pending"
	RestoreRunning   RestorePhase = "Running"
	RestoreSucceeded RestorePhase = "Succeeded"
	RestoreFailed    RestorePhase = "Failed"
	RestoreUnknown   RestorePhase = "Unknown"
)

type RestoreSessionStatus struct {
	// observedGeneration is the most recent generation observed for this resource. It corresponds to the
	// resource's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration *types.IntHash `json:"observedGeneration,omitempty"`
	Phase              RestorePhase   `json:"phase,omitempty"`
	Duration           string         `json:"duration,omitempty"`
}
