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
	"k8s.io/apimachinery/pkg/runtime"
	kmapi "kmodules.xyz/client-go/api/v1"
)

const (
	ResourceKindBundle = "Bundle"
	ResourceBundle     = "bundle"
	ResourceBundles    = "bundles"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// Bundle is the Schema for the bundles API
type Bundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BundleSpec   `json:"spec,omitempty"`
	Status BundleStatus `json:"status,omitempty"`
}

type BundleSpec struct {
	PackageDescriptor `json:",inline"`
	DisplayName       string       `json:"displayName,omitempty"`
	Features          []Feature    `json:"features,omitempty"`
	Namespace         string       `json:"namespace"`
	Packages          []PackageRef `json:"packages"`
}

type PackageRef struct {
	Chart  *ChartOption       `json:"chart,omitempty"`
	Bundle *BundleOption      `json:"bundle,omitempty"`
	OneOf  *OneOfBundleOption `json:"oneOf,omitempty"`
}

type OneOfBundleOption struct {
	Description string          `json:"description"`
	Bundles     []*BundleOption `json:"bundles,omitempty"`
}

type ChartRef struct {
	Name      string                     `json:"name"`
	SourceRef kmapi.TypedObjectReference `json:"sourceRef"`
}

type SelectionMode string

type ChartOption struct {
	ChartRef    `json:",inline"`
	Features    []string        `json:"features,omitempty"`
	Namespace   string          `json:"namespace,omitempty"`
	Versions    []VersionDetail `json:"versions"`
	MultiSelect bool            `json:"multiSelect,omitempty"`
	Required    bool            `json:"required,omitempty"`
}

type BundleRef struct {
	Name      string                     `json:"name"`
	SourceRef kmapi.TypedObjectReference `json:"sourceRef"`
}

type BundleOption struct {
	BundleRef `json:",inline"`
	Version   string `json:"version"`
}

type VersionOption struct {
	Version    string `json:"version"`
	Selected   bool   `json:"selected,omitempty"`
	ValuesFile string `json:"valuesFile,omitempty"`
	// RFC 6902 compatible json patch. ref: http://jsonpatch.com
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	ValuesPatch *runtime.RawExtension `json:"valuesPatch,omitempty"`
}

type VersionDetail struct {
	VersionOption `json:",inline"`
	Resources     *ResourceDefinitions `json:"resources,omitempty"`
	WaitFors      []WaitFlags          `json:"waitFors,omitempty"`
	// jsonpatch path in Values where the license key will be set using replace operation, if defined.
	// See: http://jsonpatch.com
	LicenseKeyPath string `json:"licenseKeyPath,omitempty"`
}

//+kubebuilder:object:root=true

// BundleList contains a list of Bundle
type BundleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bundle `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Bundle{}, &BundleList{})
}

type BundleStatus struct {
	// ObservedGeneration is the most recent generation observed for this resource. It corresponds to the
	// resource's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

type InstallOptions struct {
	ChartRef    `json:",inline"`
	Version     string `json:"version"`
	ReleaseName string `json:"releaseName"`
	Namespace   string `json:"namespace"`
}
