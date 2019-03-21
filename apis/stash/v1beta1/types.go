package v1beta1

import (
	core "k8s.io/api/core/v1"
)

// Param declares a value to use for the Param called Name.
type Param struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TaskRef struct {
	Name string `json:"name,omitempty"`
	// +optional
	Params []Param `json:"params,omitempty"`
}

type Target struct {
	// Ref refers to the target of backup/restore
	Ref TargetRef `json:"ref,omitempty"`
	// Directories specify the directories to backup
	// +optional
	Directories []string `json:"directories,omitempty"`
	// VolumeMounts specifies the volumes to mount inside stash sidecar/init container
	// Specify the volumes that contains the target directories
	// +optional
	VolumeMounts []core.VolumeMount `json:"volumeMounts,omitempty"`
}

type TargetRef struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
}
