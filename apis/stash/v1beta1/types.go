package v1beta1

type TaskRef struct {
	Name   string            `json:"name,omitempty"`
	Inputs map[string]string `json:"inputs,omitempty"`
}

type Target struct {
	// Ref refers to the target of backup/restore
	Ref TargetRef `json:"ref,omitempty"`
	// Directories specify the directories to backup
	// +optional
	Directories []string `json:"directories,omitempty"`
	// Path within the container at which the volume should be mounted.  Must
	// not contain ':'.
	// +optional
	MountPath string `json:"mountPath,omitempty"`
}

type TargetRef struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
}
