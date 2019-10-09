package v1beta1

import (
	core "k8s.io/api/core/v1"
)

// BackupInvokerRef contains information that points to the backup configuration or batch being used
type BackupInvokerRef struct {
	// APIGroup is the group for the resource being referenced
	// +optional
	APIGroup string `json:"apiGroup,omitempty"`
	// Kind is the type of resource being referenced
	Kind string `json:"kind"`
	// Name is the name of resource being referenced
	Name string `json:"name"`
}

// Param declares a value to use for the Param called Name.
type Param struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TaskRef struct {
	Name string `json:"name"`
	// +optional
	Params []Param `json:"params,omitempty"`
}

type BackupTarget struct {
	// Ref refers to the backup target
	Ref TargetRef `json:"ref,omitempty"`
	// Paths specify the file paths to backup
	// +optional
	Paths []string `json:"paths,omitempty"`
	// VolumeMounts specifies the volumes to mount inside stash sidecar/init container
	// Specify the volumes that contains the target directories
	// +optional
	VolumeMounts []core.VolumeMount `json:"volumeMounts,omitempty"`
	//replicas are the desired number of replicas whose data should be backed up.
	// If unspecified, defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// Name of the VolumeSnapshotClass used by the VolumeSnapshot. If not specified, a default snapshot class will be used if it is available.
	// Use this field only if the "driver" field is set to "volumeSnapshotter".
	// +optional
	VolumeSnapshotClassName string `json:"snapshotClassName,omitempty"`
}

type RestoreTarget struct {
	// Ref refers to the restore,target
	Ref TargetRef `json:"ref,omitempty"`
	// VolumeMounts specifies the volumes to mount inside stash sidecar/init container
	// Specify the volumes that contains the target directories
	// +optional
	VolumeMounts []core.VolumeMount `json:"volumeMounts,omitempty"`
	// replicas is the desired number of replicas of the given Template.
	// These are replicas in the sense that they are instantiations of the
	// same Template, but individual replicas also have a consistent identity.
	// If unspecified, defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`
	// volumeClaimTemplates is a list of claims that will be created while restore from VolumeSnapshot
	// +optional
	VolumeClaimTemplates []core.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`
}

type TargetRef struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
}
