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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BundleView struct {
	metav1.TypeMeta  `json:",inline"`
	BundleOptionView `json:",inline"`
	LicenseKey       string `json:"licenseKey,omitempty"`
}

type BundleOptionView struct {
	PackageMeta `json:",inline"`
	DisplayName string        `json:"displayName"`
	Features    []Feature     `json:"features,omitempty"`
	Packages    []PackageCard `json:"packages"`
}

type PackageCard struct {
	Chart  *ChartCard             `json:"chart,omitempty"`
	Bundle *BundleOptionView      `json:"bundle,omitempty"`
	OneOf  *OneOfBundleOptionView `json:"oneOf,omitempty"`
}

type OneOfBundleOptionView struct {
	Description string              `json:"description"`
	Bundles     []*BundleOptionView `json:"bundles,omitempty"`
}

type ChartCard struct {
	ChartRef          `json:",inline"`
	PackageDescriptor `json:",inline"`
	Features          []string        `json:"features,omitempty"`
	Namespace         string          `json:"namespace,omitempty"`
	Versions          []VersionOption `json:"versions"`
	MultiSelect       bool            `json:"multiSelect,omitempty"`
	Required          bool            `json:"required,omitempty"`
	Selected          bool            `json:"selected,omitempty"`
}
