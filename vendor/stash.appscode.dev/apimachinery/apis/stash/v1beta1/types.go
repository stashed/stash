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
	core "k8s.io/api/core/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

// BackupInvokerRef contains information that points to the backup configuration or batch being used
type BackupInvokerRef struct {
	// APIGroup is the group for the resource being referenced
	// +optional
	APIGroup string `json:"apiGroup,omitempty" protobuf:"bytes,1,opt,name=apiGroup"`
	// Kind is the type of resource being referenced
	Kind string `json:"kind" protobuf:"bytes,2,opt,name=kind"`
	// Name is the name of resource being referenced
	Name string `json:"name" protobuf:"bytes,3,opt,name=name"`
}

// Param declares a value to use for the Param called Name.
type Param struct {
	Name  string `json:"name" protobuf:"bytes,1,opt,name=name"`
	Value string `json:"value" protobuf:"bytes,2,opt,name=value"`
}

type TaskRef struct {
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// +optional
	Params []Param `json:"params,omitempty" protobuf:"bytes,2,rep,name=params"`
}

type BackupTarget struct {
	// Ref refers to the backup target
	Ref TargetRef `json:"ref,omitempty" protobuf:"bytes,1,opt,name=ref"`
	// Paths specify the file paths to backup
	// +optional
	Paths []string `json:"paths,omitempty" protobuf:"bytes,2,rep,name=paths"`
	// VolumeMounts specifies the volumes to mount inside stash sidecar/init container
	// Specify the volumes that contains the target directories
	// +optional
	VolumeMounts []core.VolumeMount `json:"volumeMounts,omitempty" protobuf:"bytes,3,rep,name=volumeMounts"`
	//replicas are the desired number of replicas whose data should be backed up.
	// If unspecified, defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty" protobuf:"varint,4,opt,name=replicas"`
	// Name of the VolumeSnapshotClass used by the VolumeSnapshot. If not specified, a default snapshot class will be used if it is available.
	// Use this field only if the "driver" field is set to "volumeSnapshotter".
	// +optional
	VolumeSnapshotClassName string `json:"snapshotClassName,omitempty" protobuf:"bytes,5,opt,name=snapshotClassName"`
}

type RestoreTarget struct {
	// Ref refers to the restore,target
	Ref TargetRef `json:"ref,omitempty" protobuf:"bytes,2,opt,name=ref"`
	// VolumeMounts specifies the volumes to mount inside stash sidecar/init container
	// Specify the volumes that contains the target directories
	// +optional
	VolumeMounts []core.VolumeMount `json:"volumeMounts,omitempty" protobuf:"bytes,3,rep,name=volumeMounts"`
	// replicas is the desired number of replicas of the given Template.
	// These are replicas in the sense that they are instantiations of the
	// same Template, but individual replicas also have a consistent identity.
	// If unspecified, defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`
	// volumeClaimTemplates is a list of claims that will be created while restore from VolumeSnapshot
	// +optional
	VolumeClaimTemplates []ofst.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty" protobuf:"bytes,4,rep,name=volumeClaimTemplates"`
}

type TargetRef struct {
	APIVersion string `json:"apiVersion,omitempty" protobuf:"bytes,1,opt,name=apiVersion"`
	Kind       string `json:"kind,omitempty" protobuf:"bytes,2,opt,name=kind"`
	Name       string `json:"name,omitempty" protobuf:"bytes,3,opt,name=name"`
}

// +kubebuilder:validation:Enum=BackupTargetFound;StashSidecarInjected;CronJobCreated
type BackupInvokerCondition string

const (
	// BackupTargetFound indicates whether the backup target was found
	BackupTargetFound BackupInvokerCondition = "BackupTargetFound"
	// StashSidecarInjected indicates whether stash sidecar was injected into the targeted workload
	// This condition is applicable only for sidecar model
	StashSidecarInjected BackupInvokerCondition = "StashSidecarInjected"
	// CronJobCreated indicates whether the backup triggering CronJob was created
	CronJobCreated BackupInvokerCondition = "CronJobCreated"
)

// +kubebuilder:validation:Enum=RestoreTargetFound;StashInitContainerInjected;RestoreJobCreated
type RestoreSessionCondition string

const (
	// RestoreTargetFound indicates whether the restore target was found
	RestoreTargetFound RestoreSessionCondition = "RestoreTargetFound"
	// StashInitContainerInjected indicates whether stash init-container was injected into the targeted workload
	// This condition is applicable only for sidecar model
	StashInitContainerInjected RestoreSessionCondition = "StashInitContainerInjected"
	// RestoreJobCreated indicates whether the restore job was created
	RestoreJobCreated RestoreSessionCondition = "RestoreJobCreated"
)
