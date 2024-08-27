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

package shared

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type BootstrapPresets struct {
	// +optional
	OfflineInstaller bool              `json:"offlineInstaller"`
	Image            ImageRegistrySpec `json:"image"`
	Registry         RegistryInfo      `json:"registry"`
	Helm             HelmInfo          `json:"helm"`
}

type ImageRegistrySpec struct {
	//+optional
	Proxies RegistryProxies `json:"proxies"`
}

type RegistryProxies struct {
	// company/bin:1.23
	//+optional
	DockerHub string `json:"dockerHub"`
	// alpine, nginx etc.
	//+optional
	DockerLibrary string `json:"dockerLibrary"`
	// ghcr.io
	//+optional
	GHCR string `json:"ghcr"`
	// quay.io
	//+optional
	Quay string `json:"quay"`
	// registry.k8s.io
	//+optional
	Kubernetes string `json:"kubernetes"`
	// r.appscode.com
	//+optional
	AppsCode string `json:"appscode"`
}

type RegistryInfo struct {
	//+optional
	Credentials map[string]string `json:"credentials"`
	//+optional
	Certs map[string]string `json:"certs"`
	//+optional
	ImagePullSecrets []string `json:"imagePullSecrets"`
}

type HelmInfo struct {
	CreateNamespace bool                       `json:"createNamespace"`
	Repositories    map[string]*HelmRepository `json:"repositories"`
	Releases        map[string]*HelmRelease    `json:"releases"`
}

type HelmRepository struct {
	// URL of the Helm repository, a valid URL contains at least a protocol and
	// host.
	// +required
	URL string `json:"url"`

	// SecretRef specifies the Secret containing authentication credentials
	// for the HelmRepository.
	// For HTTP/S basic auth the secret must contain 'username' and 'password'
	// fields.
	// For TLS the secret must contain a 'certFile' and 'keyFile', and/or
	// 'caFile' fields.
	// +optional
	SecretName string `json:"secretName,omitempty"`

	// Interval at which to check the URL for updates.
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern="^([0-9]+(\\.[0-9]+)?(ms|s|m|h))+$"
	// +optional
	Interval *metav1.Duration `json:"interval,omitempty"`

	// The timeout of index downloading, defaults to 60s.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Type of the HelmRepository.
	// When this field is set to  "oci", the URL field value must be prefixed with "oci://".
	// +kubebuilder:validation:Enum=default;oci
	// +optional
	Type string `json:"type,omitempty"`

	// Provider used for authentication, can be 'aws', 'azure', 'gcp' or 'generic'.
	// This field is optional, and only taken into account if the .spec.type field is set to 'oci'.
	// When not specified, defaults to 'generic'.
	// +kubebuilder:validation:Enum=generic;aws;azure;gcp
	// +kubebuilder:default:=generic
	// +optional
	Provider string `json:"provider,omitempty"`
}

type HelmRelease struct {
	Enabled bool                  `json:"enabled"`
	Version string                `json:"version"`
	Values  *runtime.RawExtension `json:"values,omitempty"`
}
