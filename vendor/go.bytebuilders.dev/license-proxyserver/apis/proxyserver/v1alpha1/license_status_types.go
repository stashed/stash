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
	licenseapi "go.bytebuilders.dev/license-verifier/apis/licenses/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceKindLicenseStatus = "LicenseStatus"
	ResourceLicenseStatus     = "licensestatus"
	ResourceLicenseStatuses   = "licensestatuses"
)

// LicenseStatusSpec defines the desired state of License
type LicenseStatusSpec struct {
	Feature            string `json:"feature"`
	ServiceAccountName string `json:"serviceAccountName"`
}

// LicenseStatusStatus defines the status of License
type LicenseStatusStatus struct {
	License licenseapi.License `json:"license"`
}

// +genclient
// +genclient:nonNamespaced
// +genclient:onlyVerbs=get,list
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// LicenseStatus is the Schema for the licensestatuses API
type LicenseStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LicenseStatusSpec   `json:"spec,omitempty"`
	Status LicenseStatusStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:object:root=true

// LicenseStatusList contains a list of LicenseStatus
type LicenseStatusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LicenseStatus `json:"items"`
}
