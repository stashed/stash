/*
Copyright The Stash Authors.

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

package v1alpha1

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	store "kmodules.xyz/objectstore-api/api/v1"
)

const (
	ResourceKindRecovery     = "Recovery"
	ResourceSingularRecovery = "recovery"
	ResourcePluralRecovery   = "recoveries"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=recoveries,singular=recovery,shortName=rec,categories={storage,appscode,all}
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Repository-NS",type="string",JSONPath=".spec.repository.namespace"
// +kubebuilder:printcolumn:name="Repository-Name",type="string",JSONPath=".spec.repository.name"
// +kubebuilder:printcolumn:name="Snapshot",type="string",JSONPath=".spec.snapshot"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Recovery struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              RecoverySpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status            RecoveryStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type RecoverySpec struct {
	Repository core.ObjectReference `json:"repository" protobuf:"bytes,1,opt,name=repository"`
	// Snapshot to recover. Default is latest snapshot.
	// +optional
	Snapshot         string                      `json:"snapshot,omitempty" protobuf:"bytes,2,opt,name=snapshot"`
	Paths            []string                    `json:"paths,omitempty" protobuf:"bytes,3,rep,name=paths"`
	RecoveredVolumes []store.LocalSpec           `json:"recoveredVolumes,omitempty" protobuf:"bytes,4,rep,name=recoveredVolumes"`
	ImagePullSecrets []core.LocalObjectReference `json:"imagePullSecrets,omitempty" protobuf:"bytes,5,rep,name=imagePullSecrets"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	NodeSelector map[string]string `json:"nodeSelector,omitempty" protobuf:"bytes,6,rep,name=nodeSelector"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RecoveryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Recovery `json:"items,omitempty" protobuf:"bytes,2,rep,name=items"`
}

// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed;Unknown
type RecoveryPhase string

const (
	RecoveryPending   RecoveryPhase = "Pending"
	RecoveryRunning   RecoveryPhase = "Running"
	RecoverySucceeded RecoveryPhase = "Succeeded"
	RecoveryFailed    RecoveryPhase = "Failed"
	RecoveryUnknown   RecoveryPhase = "Unknown"
)

type RecoveryStatus struct {
	// observedGeneration is the most recent generation observed for this resource. It corresponds to the
	// resource's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64          `json:"observedGeneration,omitempty" protobuf:"varint,1,opt,name=observedGeneration"`
	Phase              RecoveryPhase  `json:"phase,omitempty" protobuf:"bytes,2,opt,name=phase,casttype=RecoveryPhase"`
	Stats              []RestoreStats `json:"stats,omitempty" protobuf:"bytes,3,rep,name=stats"`
}

type RestoreStats struct {
	Path     string        `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
	Phase    RecoveryPhase `json:"phase,omitempty" protobuf:"bytes,2,opt,name=phase,casttype=RecoveryPhase"`
	Duration string        `json:"duration,omitempty" protobuf:"bytes,3,opt,name=duration"`
}
