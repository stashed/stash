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
	kmapi "kmodules.xyz/client-go/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ResourcePanel struct {
	metav1.TypeMeta `json:",inline"`
	Sections        []*PanelSection `json:"sections,omitempty"`
}

type PanelSection struct {
	Name              string `json:"name,omitempty"`
	ResourceClassInfo `json:",inline"`
	Weight            int          `json:"-"`
	Entries           []PanelEntry `json:"entries"`
}

type PanelEntry struct {
	Name string `json:"name"`
	// +optional
	Path     string            `json:"path,omitempty"`
	Resource *kmapi.ResourceID `json:"resource,omitempty"`
	// +optional
	Required bool `json:"required,omitempty"`
	// +optional
	Icons      []ImageSpec           `json:"icons,omitempty"`
	Namespaced bool                  `json:"namespaced"`
	Missing    bool                  `json:"missing,omitempty"`
	Installer  *DeploymentParameters `json:"installer,omitempty"`
}
