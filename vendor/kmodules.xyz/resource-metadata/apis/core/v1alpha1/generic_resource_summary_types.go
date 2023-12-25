/*
Copyright AppsCode Inc. and Contributors.

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
	ResourceKindResourceSummary = "ResourceSummary"
	ResourceResourceSummary     = "resourcesummary"
	ResourceResourceSummaries   = "resourcesummaries"
)

type ResourceSummarySpec struct {
	Cluster       kmapi.ClusterMetadata     `json:"cluster,omitempty"`
	APIType       kmapi.ResourceID          `json:"apiType"`
	TotalResource core.ResourceRequirements `json:"totalResource,omitempty"`
	AppResource   core.ResourceRequirements `json:"appResource,omitempty"`
	Count         int                       `json:"count,omitempty"`
}

type KubernetesInfo struct {
	// https://github.com/kmodules/client-go/blob/master/tools/clusterid/lib.go
	ClusterName  string            `json:"clusterName,omitempty"`
	ClusterUID   string            `json:"clusterUID,omitempty"`
	Version      *version.Info     `json:"version,omitempty"`
	ControlPlane *ControlPlaneInfo `json:"controlPlane,omitempty"`
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

// ResourceSummary is the Schema for the ResourceSummary API

// +genclient
// +genclient:onlyVerbs=get,list
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ResourceSummary struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ResourceSummarySpec `json:"spec,omitempty"`
}

// ResourceSummaryList contains a list of ResourceSummary

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ResourceSummaryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceSummary `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourceSummary{}, &ResourceSummaryList{})
}
