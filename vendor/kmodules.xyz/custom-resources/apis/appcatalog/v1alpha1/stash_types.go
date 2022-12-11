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

// StashAddon defines a Stash backup and restore task definitions.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type StashAddon struct {
	metav1.TypeMeta `json:",inline,omitempty"`
	Stash           StashAddonSpec `json:"stash,omitempty"`
}

// StashAddonSpec is the spec for app
type StashAddonSpec struct {
	Addon StashTaskSpec `json:"addon,omitempty"`
}

// StashTaskSpec is the spec for app
type StashTaskSpec struct {
	// Backup task definition
	BackupTask TaskRef `json:"backupTask"`

	// Restore task definition
	RestoreTask TaskRef `json:"restoreTask"`
}

type TaskRef struct {
	Name string `json:"name"`
	// Params specifies a list of parameter to pass to the Task. Stash will use this parameters to resolve the task.
	// +optional
	Params []Param `json:"params,omitempty"`
}

// Param declares a value to use for the Param called Name.
type Param struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
