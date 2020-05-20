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
	ResourceKindRestic     = "Restic"
	ResourceSingularRestic = "restic"
	ResourcePluralRestic   = "restics"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=restics,singular=restic,shortName=rst,categories={stash,appscode,all}
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Selector",type="string",JSONPath=".spec.selector"
// +kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=".spec.schedule"
// +kubebuilder:printcolumn:name="Backup-Type",type="string",JSONPath=".spec.type",priority=10
// +kubebuilder:printcolumn:name="Paused",type="boolean",JSONPath=".spec.paused"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Restic struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              ResticSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

type ResticSpec struct {
	Selector   metav1.LabelSelector `json:"selector,omitempty" protobuf:"bytes,1,opt,name=selector"`
	FileGroups []FileGroup          `json:"fileGroups,omitempty" protobuf:"bytes,2,rep,name=fileGroups"`
	Backend    store.Backend        `json:"backend,omitempty" protobuf:"bytes,3,opt,name=backend"`
	Schedule   string               `json:"schedule,omitempty" protobuf:"bytes,4,opt,name=schedule"`
	// Pod volumes to mount into the sidecar container's filesystem.
	VolumeMounts []core.VolumeMount `json:"volumeMounts,omitempty" protobuf:"bytes,5,rep,name=volumeMounts"`
	// Compute Resources required by the sidecar container.
	Resources         core.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,6,opt,name=resources"`
	RetentionPolicies []RetentionPolicy         `json:"retentionPolicies,omitempty" protobuf:"bytes,7,rep,name=retentionPolicies"`
	// https://github.com/stashed/stash/issues/225
	Type BackupType `json:"type,omitempty" protobuf:"bytes,8,opt,name=type,casttype=BackupType"`
	//Indicates that the Restic is paused from taking backup. Default value is 'false'
	// +optional
	Paused bool `json:"paused,omitempty" protobuf:"varint,9,opt,name=paused"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use. For example,
	// in the case of docker, only DockerConfig type secrets are honored.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	ImagePullSecrets []core.LocalObjectReference `json:"imagePullSecrets,omitempty" protobuf:"bytes,10,rep,name=imagePullSecrets"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ResticList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Restic `json:"items,omitempty" protobuf:"bytes,2,rep,name=items"`
}

type FileGroup struct {
	// Source of the backup volumeName:path
	Path string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
	// Tags of a snapshots
	Tags []string `json:"tags,omitempty" protobuf:"bytes,2,rep,name=tags"`
	// retention policy of snapshots
	RetentionPolicyName string `json:"retentionPolicyName,omitempty" protobuf:"bytes,3,opt,name=retentionPolicyName"`
}

// +kubebuilder:validation:Enum=online;offline
type BackupType string

const (
	BackupOnline  BackupType = "online"  // default, injects sidecar
	BackupOffline BackupType = "offline" // injects init container
)

// +kubebuilder:validation:Enum=--keep-last;--keep-hourly;--keep-daily;--keep-weekly;--keep-monthly;--keep-yearly;--keep-tag
type RetentionStrategy string

const (
	KeepLast    RetentionStrategy = "--keep-last"
	KeepHourly  RetentionStrategy = "--keep-hourly"
	KeepDaily   RetentionStrategy = "--keep-daily"
	KeepWeekly  RetentionStrategy = "--keep-weekly"
	KeepMonthly RetentionStrategy = "--keep-monthly"
	KeepYearly  RetentionStrategy = "--keep-yearly"
	KeepTag     RetentionStrategy = "--keep-tag"
)

type RetentionPolicy struct {
	Name        string   `json:"name" protobuf:"bytes,1,opt,name=name"`
	KeepLast    int64    `json:"keepLast,omitempty" protobuf:"varint,2,opt,name=keepLast"`
	KeepHourly  int64    `json:"keepHourly,omitempty" protobuf:"varint,3,opt,name=keepHourly"`
	KeepDaily   int64    `json:"keepDaily,omitempty" protobuf:"varint,4,opt,name=keepDaily"`
	KeepWeekly  int64    `json:"keepWeekly,omitempty" protobuf:"varint,5,opt,name=keepWeekly"`
	KeepMonthly int64    `json:"keepMonthly,omitempty" protobuf:"varint,6,opt,name=keepMonthly"`
	KeepYearly  int64    `json:"keepYearly,omitempty" protobuf:"varint,7,opt,name=keepYearly"`
	KeepTags    []string `json:"keepTags,omitempty" protobuf:"bytes,8,rep,name=keepTags"`
	Prune       bool     `json:"prune" protobuf:"varint,9,opt,name=prune"`
	DryRun      bool     `json:"dryRun,omitempty" protobuf:"varint,10,opt,name=dryRun"`
}
