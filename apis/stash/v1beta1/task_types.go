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
// +kubebuilder:resource:path=tasks,singular=task,scope=Cluster,shortName=task,categories={stash,appscode}
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              TaskSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

type TaskSpec struct {
	Steps []FunctionRef `json:"steps,omitempty" protobuf:"bytes,1,rep,name=steps"`
	// List of volumes that can be mounted by containers belonging to the pod created for this task.
	// +optional
	Volumes []core.Volume `json:"volumes,omitempty" protobuf:"bytes,2,rep,name=volumes"`
}

type FunctionRef struct {
	// Name indicates the name of Function crd
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// Inputs specifies the inputs of respective Function
	// +optional
	Params []Param `json:"params,omitempty" protobuf:"bytes,2,rep,name=params"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Task `json:"items,omitempty" protobuf:"bytes,2,rep,name=items"`
}
