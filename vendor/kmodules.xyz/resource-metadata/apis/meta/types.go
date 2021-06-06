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

package meta

import (
	kmapi "kmodules.xyz/client-go/api/v1"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ReferenceType string

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ResourceDescriptor struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Spec ResourceDescriptorSpec
}

type ResourceDescriptorSpec struct {
	Resource    kmapi.ResourceID
	Columns     []ResourceColumnDefinition
	SubTables   []ResourceSubTableDefinition
	Connections []ResourceConnection
	Pages       []RelatedResourcePage
	Status      *StatusCodes

	Validation *apiextensions.CustomResourceValidation

	Icons       []ImageSpec
	Maintainers []ContactData
	Links       []Link

	ResourceRequirements []ResourceRequirements

	UI *UIParameters

	Installer *DeploymentParameters
}

type ResourceRequirements struct {
	Units     string
	Shards    string
	Resources string
}

type RelatedResourcePage struct {
	Name      string
	Resources []ResourceSection
}

type ResourceSection struct {
	Ref         GroupVersionResource
	DisplayMode ResourceDisplayMode
	Actions     ResourceActions
}

type ResourceDisplayMode string

const (
	DisplayModeList  = "List"
	DisplayModeField = "Field"
)

type ResourceActions struct {
	Create ResourceAction
}

type ResourceAction string

const (
	ActionNever   = "Never"
	ActionAlways  = "Always"
	ActionIfEmpty = "IfEmpty"
)

type StatusCodes struct {
	Success []string
	Danger  []string
	Warning []string
}

type UIParameters struct {
	Options            *ChartRepoRef
	Editor             *ChartRepoRef
	InstanceLabelPaths []string
}

type DeploymentParameters struct {
	ProductID string
	PlanID    string
	Chart     *ChartRepoRef
}

type ChartRepoRef struct {
	Name    string
	URL     string
	Version string
}

type ConnectionType string

const (
	MatchSelector ConnectionType = "MatchSelector"
	MatchName     ConnectionType = "MatchName"
	MatchRef      ConnectionType = "MatchRef"
	OwnedBy       ConnectionType = "OwnedBy"
)

type ResourceConnection struct {
	Target metav1.TypeMeta
	ResourceConnectionSpec
}

type ResourceConnectionSpec struct {
	Type            ConnectionType
	NamespacePath   string
	TargetLabelPath string
	SelectorPath    string
	Selector        *metav1.LabelSelector
	NameTemplate    string
	References      []string
	Level           OwnershipLevel
}

type OwnershipLevel string

const (
	Reference  OwnershipLevel = ""
	Owner      OwnershipLevel = "Owner"
	Controller OwnershipLevel = "Controller"
)

// ResourceColumnDefinition specifies a column for server side printing.
type ResourceColumnDefinition struct {
	// name is a human readable name for the column.
	Name string
	// type is an OpenAPI type definition for this column.
	// See https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#data-types for more.
	Type string
	// format is an optional OpenAPI type definition for this column. The 'name' format is applied
	// to the primary identifier column to assist in clients identifying column is the resource name.
	// See https://github.com/OAI/OpenAPI-Specification/blob/master/versions/2.0.md#data-types for more.
	Format string
	// description is a human readable description of this column.
	Description string
	// priority is an integer defining the relative importance of this column compared to others. Lower
	// numbers are considered higher priority. Columns that may be omitted in limited space scenarios
	// should be given a higher priority.
	Priority int32
	// PathTemplate is a Go text template that will be evaluated to determine cell value.
	PathTemplate string
}

type ResourceSubTableDefinition struct {
	Name      string
	FieldPath string
	Columns   []ResourceColumnDefinition
}

type ImageSpec struct {
	Source string
	Size   string
	Type   string
}

type ContactData struct {
	Name  string
	URL   string
	Email string
}

type Link struct {
	Description string
	URL         string
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceDescriptorList is a list of ResourceDescriptor objects.
type ResourceDescriptorList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []ResourceDescriptor
}

type GroupResources struct {
	Groups   []string
	Resource string
}

type GroupVersionResource struct {
	Group    string
	Version  string
	Resource string
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PathFinder struct {
	metav1.TypeMeta
	Request  *PathRequest
	Response *PathResponse
}

type PathRequest struct {
	Source GroupVersionResource
	Target *GroupVersionResource
}

type PathResponse struct {
	Paths []Path
}

type Path struct {
	Source   GroupVersionResource
	Target   GroupVersionResource
	Distance uint64
	Edges    []*Edge
}

type Edge struct {
	Src        GroupVersionResource
	Dst        GroupVersionResource
	W          uint64
	Connection ResourceConnectionSpec
	Forward    bool
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type GraphFinder struct {
	metav1.TypeMeta
	Request  *GraphRequest
	Response *GraphResponse
}

type GraphRequest struct {
	Source GroupVersionResource
}

type GraphResponse struct {
	Source      GroupVersionResource
	Connections []*Edge
}

type Table struct {
	metav1.TypeMeta
	metav1.ListMeta
	ColumnDefinitions []ResourceColumnDefinition
	Rows              []TableRow

	SubTables []SubTable
}

type SubTable struct {
	Name              string
	ColumnDefinitions []ResourceColumnDefinition
	Rows              []TableRow
}

type TableRow struct {
	Cells []interface{}
}

type IncludeObjectPolicy string

const (
	IncludeNone     IncludeObjectPolicy = "None"
	IncludeMetadata IncludeObjectPolicy = "Metadata"
	IncludeObject   IncludeObjectPolicy = "Object"
)

type TableOptions struct {
	metav1.TypeMeta
	NoHeaders     bool
	IncludeObject IncludeObjectPolicy
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ResourceClass struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Spec ResourceClassSpec
}

type ResourceClassSpec struct {
	ResourceClassInfo
	Weight  int
	Entries []Entry
}

type ResourceClassInfo struct {
	APIGroup    string
	Icons       []ImageSpec
	Maintainers []ContactData
	Links       []Link
}

type Entry struct {
	Name     string
	Path     string
	Type     *GroupResources
	Required bool
	Icons    []ImageSpec
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ResourceClassList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []ResourceClass
}
