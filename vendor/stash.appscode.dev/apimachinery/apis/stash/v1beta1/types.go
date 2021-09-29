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
	// Name specifies the name of the Task to use for backup/restore purpose. If your database has been deployed with KubeDB,
	// then keep this field empty. Stash will read the Task info from the respective AppBinding.
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// Params specifies a list of parameter to pass to the Task. Stash will use this parameters to resolve the task.
	// +optional
	Params []Param `json:"params,omitempty" protobuf:"bytes,2,rep,name=params"`
}

type BackupTarget struct {
	// Alias represents the identifier of the backed up data in the repository.
	// This will be used as `hostname` or will be used to generate the `hostname` for the restic repository.
	// +optional
	Alias string `json:"alias,omitempty" protobuf:"bytes,1,opt,name=alias"`
	// Ref refers to the backup target
	Ref TargetRef `json:"ref,omitempty" protobuf:"bytes,2,opt,name=ref"`
	// Paths specify the file paths to backup
	// +optional
	Paths []string `json:"paths,omitempty" protobuf:"bytes,3,rep,name=paths"`
	// VolumeMounts specifies the volumes to mount inside stash sidecar/init container
	// Specify the volumes that contains the target directories
	// +optional
	VolumeMounts []core.VolumeMount `json:"volumeMounts,omitempty" protobuf:"bytes,4,rep,name=volumeMounts"`
	//replicas are the desired number of replicas whose data should be backed up.
	// If unspecified, defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty" protobuf:"varint,5,opt,name=replicas"`
	// Name of the VolumeSnapshotClass used by the VolumeSnapshot. If not specified, a default snapshot class will be used if it is available.
	// Use this field only if the "driver" field is set to "volumeSnapshotter".
	// +optional
	VolumeSnapshotClassName string `json:"snapshotClassName,omitempty" protobuf:"bytes,6,opt,name=snapshotClassName"`
	// Exclude specifies a list of patterns for the files to ignore during backup.
	// Stash will ignore those files that match the specified patterns.
	// Supported only for "Restic" driver
	// +optional
	Exclude []string `json:"exclude,omitempty" protobuf:"bytes,7,rep,name=exclude"`
	// Args specifies a list of arguments to pass to the backup driver.
	// +optional
	Args []string `json:"args,omitempty" protobuf:"bytes,8,rep,name=args"`
}

type RestoreTarget struct {
	// Alias represents the identifier of the backed up data in the repository.
	// This will be used as `sourceHost` and `targetHosts` or will be used to generate them.
	// +optional
	Alias string `json:"alias,omitempty" protobuf:"bytes,1,opt,name=alias"`
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
	Replicas *int32 `json:"replicas,omitempty" protobuf:"varint,4,opt,name=replicas"`
	// volumeClaimTemplates is a list of claims that will be created while restore from VolumeSnapshot
	// +optional
	VolumeClaimTemplates []ofst.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty" protobuf:"bytes,5,rep,name=volumeClaimTemplates"`
	// Rules specifies different restore options for different hosts
	// +optional
	Rules []Rule `json:"rules,omitempty" protobuf:"bytes,6,rep,name=rules"`
	// Args specifies a list of arguments to pass to the restore driver.
	// +optional
	Args []string `json:"args,omitempty" protobuf:"bytes,7,rep,name=args"`
}

type Rule struct {
	// Subjects specifies the list of hosts that are subject to this rule
	// +optional
	TargetHosts []string `json:"targetHosts,omitempty" protobuf:"bytes,1,rep,name=targetHosts"`
	// SourceHost specifies the name of the host whose backed up state we are trying to restore
	// By default, it will indicate the workload itself
	// +optional
	SourceHost string `json:"sourceHost,omitempty" protobuf:"bytes,2,opt,name=sourceHost"`
	// Snapshots specifies the list of snapshots that will be restored for the host under this rule.
	// Don't specify if you have specified paths field.
	// +optional
	Snapshots []string `json:"snapshots,omitempty" protobuf:"bytes,3,rep,name=snapshots"`
	// Paths specifies the paths to be restored for the hosts under this rule.
	// Don't specify if you have specified snapshots field.
	// +optional
	Paths []string `json:"paths,omitempty" protobuf:"bytes,4,rep,name=paths"`
	// Exclude specifies a list of patterns for the files to ignore during restore.
	// Stash will only restore the files that does not match those patterns.
	// Supported only for "Restic" driver
	// +optional
	Exclude []string `json:"exclude,omitempty" protobuf:"bytes,5,rep,name=exclude"`
	// Include specifies a list of patterns for the files to restore.
	// Stash will only restore the files that match those patterns.
	// Supported only for "Restic" driver
	// +optional
	Include []string `json:"include,omitempty" protobuf:"bytes,6,rep,name=include"`
}

type TargetRef struct {
	APIVersion string `json:"apiVersion,omitempty" protobuf:"bytes,1,opt,name=apiVersion"`
	Kind       string `json:"kind,omitempty" protobuf:"bytes,2,opt,name=kind"`
	Name       string `json:"name,omitempty" protobuf:"bytes,3,opt,name=name"`
}

type ExecutionOrder string

const (
	Parallel   ExecutionOrder = "Parallel"
	Sequential ExecutionOrder = "Sequential"
)
