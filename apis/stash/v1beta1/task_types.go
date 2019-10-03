package v1beta1

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceKindTask     = "Task"
	ResourcePluralTask   = "tasks"
	ResourceSingularTask = "task"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=tasks,singular=task,shortName=task,categories={stash,appscode}
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TaskSpec `json:"spec,omitempty"`
}

type TaskSpec struct {
	Steps []FunctionRef `json:"steps,omitempty"`
	// List of volumes that can be mounted by containers belonging to the pod created for this task.
	// +optional
	Volumes []core.Volume `json:"volumes,omitempty"`
}

type FunctionRef struct {
	// Name indicates the name of Function crd
	Name string `json:"name,omitempty"`
	// Inputs specifies the inputs of respective Function
	// +optional
	Params []Param `json:"params,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Task `json:"items,omitempty"`
}
