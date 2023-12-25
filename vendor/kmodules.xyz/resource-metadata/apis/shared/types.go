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
	releasesapi "x-helm.dev/apimachinery/apis/releases/v1alpha1"
	helmshared "x-helm.dev/apimachinery/apis/shared"
)

type SourceLocator struct {
	// +optional
	Resource kmapi.ResourceID `json:"resource"`
	// +optional
	Ref kmapi.ObjectReference `json:"ref"`
}

type DeploymentParameters struct {
	ProductID string                      `json:"productID,omitempty"`
	PlanID    string                      `json:"planID,omitempty"`
	Chart     *releasesapi.ChartSourceRef `json:"chart,omitempty"`
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
	Options *releasesapi.ChartSourceRef `json:"options,omitempty"`
	Editor  *releasesapi.ChartSourceRef `json:"editor,omitempty"`
	// +optional
	Actions []*ActionGroup `json:"actions,omitempty"`
	// app.kubernetes.io/instance label must be updated at these paths when refilling metadata
	// +optional
	InstanceLabelPaths []string `json:"instanceLabelPaths,omitempty"`
}

type UIParameterTemplate struct {
	Options      *releasesapi.ChartSourceRef `json:"options,omitempty"`
	Editor       *releasesapi.ChartSourceRef `json:"editor,omitempty"`
	EnforceQuota bool                        `json:"enforceQuota"`
	// +optional
	Actions []*ActionTemplateGroup `json:"actions,omitempty"`
	// app.kubernetes.io/instance label must be updated at these paths when refilling metadata
	// +optional
	InstanceLabelPaths []string `json:"instanceLabelPaths,omitempty"`
}

type ActionGroup struct {
	ActionInfo `json:",inline,omitempty"`
	Items      []Action `json:"items"`
}

type Action struct {
	ActionInfo `json:",inline,omitempty"`
	// +optional
	Icons       []helmshared.ImageSpec      `json:"icons,omitempty"`
	OperationID string                      `json:"operationId"`
	Flow        string                      `json:"flow"`
	Disabled    bool                        `json:"disabled"`
	Editor      *releasesapi.ChartSourceRef `json:"editor,omitempty"`
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

type ActionTemplateGroup struct {
	ActionInfo `json:",inline,omitempty"`
	Items      []ActionTemplate `json:"items"`
}

type ActionTemplate struct {
	ActionInfo `json:",inline,omitempty"`
	// +optional
	Icons            []helmshared.ImageSpec      `json:"icons,omitempty"`
	OperationID      string                      `json:"operationId"`
	Flow             string                      `json:"flow"`
	DisabledTemplate string                      `json:"disabledTemplate,omitempty"`
	Editor           *releasesapi.ChartSourceRef `json:"editor,omitempty"`
	EnforceQuota     bool                        `json:"enforceQuota"`
}

type ActionInfo struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}
