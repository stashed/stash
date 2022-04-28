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

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
)

const (
	ResourceKindSiteInfo = "SiteInfo"
	ResourceSiteInfo     = "siteinfo"
	ResourceSiteInfos    = "siteinfos"
)

// SiteInfo captures information of a product deployment site.

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=siteinfos,singular=siteinfo,scope=Cluster,categories={auditor,appscode,all}
type SiteInfo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Product           *ProductInfo    `json:"product,omitempty"`
	Kubernetes        *KubernetesInfo `json:"kubernetes,omitempty"`
}

type Version struct {
	Version         string `json:"version,omitempty"`
	VersionStrategy string `json:"versionStrategy,omitempty"`
	CommitHash      string `json:"commitHash,omitempty"`
	GitBranch       string `json:"gitBranch,omitempty"`
	GitTag          string `json:"gitTag,omitempty"`
	CommitTimestamp string `json:"commitTimestamp,omitempty"`
	GoVersion       string `json:"goVersion,omitempty"`
	Compiler        string `json:"compiler,omitempty"`
	Platform        string `json:"platform,omitempty"`
}

type ProductInfo struct {
	Version   Version `json:"version"`
	LicenseID string  `json:"licenseID,omitempty"`

	ProductOwnerName string `json:"productOwnerName,omitempty"`
	ProductOwnerUID  string `json:"productOwnerUID,omitempty"`

	// This has been renamed to Features
	ProductName string `json:"productName,omitempty"`
	ProductUID  string `json:"productUID,omitempty"`
}

type KubernetesInfo struct {
	// Deprecated
	ClusterName string `json:"clusterName,omitempty"`
	// Deprecated
	ClusterUID   string                 `json:"clusterUID,omitempty"`
	Cluster      *kmapi.ClusterMetadata `json:"cluster,omitempty"`
	Version      *version.Info          `json:"version,omitempty"`
	ControlPlane *ControlPlaneInfo      `json:"controlPlane,omitempty"`
	NodeStats    NodeStats              `json:"nodeStats"`
}

// https://github.com/kmodules/client-go/blob/kubernetes-1.16.3/tools/analytics/analytics.go#L66
type ControlPlaneInfo struct {
	DNSNames       []string    `json:"dnsNames,omitempty"`
	EmailAddresses []string    `json:"emailAddresses,omitempty"`
	IPAddresses    []string    `json:"ipAddresses,omitempty"`
	URIs           []string    `json:"uris,omitempty"`
	NotBefore      metav1.Time `json:"notBefore"`
	NotAfter       metav1.Time `json:"notAfter"`
}

type NodeStats struct {
	Count int `json:"count,omitempty"`

	// Capacity represents the total resources of a node.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#capacity
	// +optional
	Capacity core.ResourceList `json:"capacity,omitempty"`

	// Allocatable represents the resources of a node that are available for scheduling.
	// Defaults to Capacity.
	// +optional
	Allocatable core.ResourceList `json:"allocatable,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SiteInfoList is a list of SiteInfo
type SiteInfoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SiteInfo `json:"items,omitempty"`
}
