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
	kmapi "kmodules.xyz/client-go/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImageSpec contains information about an image used as an icon.
type ImageSpec struct {
	// The source for image represented as either an absolute URL to the image or a Data URL containing
	// the image. Data URLs are defined in RFC 2397.
	Source string `json:"src"`

	// (optional) The size of the image in pixels (e.g., 25x25).
	Size string `json:"size,omitempty"`

	// (optional) The mine type of the image (e.g., "image/png").
	Type string `json:"type,omitempty"`
}

type DeploymentParameters struct {
	ProductID string        `json:"productID,omitempty"`
	PlanID    string        `json:"planID,omitempty"`
	Chart     *ChartRepoRef `json:"chart,omitempty"`
}

// ChartRepoRef references to a single version of a Chart
type ChartRepoRef struct {
	// +optional
	URL     string `json:"url,omitempty"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ResourceLocator struct {
	Ref   metav1.GroupKind `json:"ref"`
	Query ResourceQuery    `json:"query"`
}

// +kubebuilder:validation:Enum=REST;GraphQL
type QueryType string

const (
	RESTQuery    QueryType = "REST"
	GraphQLQuery QueryType = "GraphQL"
)

type ResourceQuery struct {
	Type    QueryType       `json:"type"`
	ByLabel kmapi.EdgeLabel `json:"byLabel,omitempty"`
	Raw     string          `json:"raw,omitempty"`
}

type UIParameters struct {
	Options *ChartRepoRef `json:"options,omitempty"`
	Editor  *ChartRepoRef `json:"editor,omitempty"`
	// app.kubernetes.io/instance label must be updated at these paths when refilling metadata
	// +optional
	InstanceLabelPaths []string `json:"instanceLabelPaths,omitempty"`
}

// +kubebuilder:validation:Enum=Source;Target
type DashboardVarType string

const (
	DashboardVarTypeSource DashboardVarType = "Source"
	DashboardVarTypeTarget DashboardVarType = "Target"
)

type DashboardVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	// +optional
	// +kubebuilder:default:=Source
	Type DashboardVarType `json:"type,omitempty"`
}

type Dashboard struct {
	// +optional
	Title string `json:"title,omitempty"`
	// +optional
	Vars []DashboardVar `json:"vars,omitempty"`
	// +optional
	Panels []string `json:"panels,omitempty"`
	// +optional
	If *If `json:"if,omitempty"`
}

type If struct {
	Condition string           `json:"condition,omitempty"`
	Connected *ResourceLocator `json:"connected,omitempty"`
}
