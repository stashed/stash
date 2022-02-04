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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceKindSnapshot     = "Snapshot"
	ResourcePluralSnapshot   = "snapshots"
	ResourceSingularSnapshot = "snapshot"
)

// +genclient
// +genclient:skipVerbs=create,update,patch,deleteCollection,watch
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="Repository",type="string",JSONPath=".status.repository"
// +kubebuilder:printcolumn:name="Hostname",type="string",JSONPath=".status.hostname"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".uid"
type Snapshot struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            SnapshotStatus `json:"status,omitempty"`
}

type SnapshotStatus struct {
	Tree       string   `json:"tree"`
	Paths      []string `json:"paths"`
	Hostname   string   `json:"hostname"`
	Username   string   `json:"username"`
	UID        int32    `json:"uid"`
	Gid        int32    `json:"gid"`
	Tags       []string `json:"tags,omitempty"`
	Repository string   `json:"repository"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type SnapshotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Snapshot `json:"items"`
}
