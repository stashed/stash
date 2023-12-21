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
	crdv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"x-helm.dev/apimachinery/apis/shared"
)

type PackageDescriptor struct {
	// Type is the type of the application (e.g. WordPress, MySQL, Cassandra).
	Type string `json:"type,omitempty"`

	// Description is a brief string description of the Application.
	Description string `json:"description,omitempty"`

	// Icons is an optional list of icons for an application. Icon information includes the source, size,
	// and mime type.
	Icons []shared.ImageSpec `json:"icons,omitempty"`

	// Maintainers is an optional list of maintainers of the application. The maintainers in this list maintain the
	// the source code, images, and package for the application.
	Maintainers []shared.ContactData `json:"maintainers,omitempty"`

	// Keywords is an optional list of key words associated with the application (e.g. MySQL, RDBMS, database).
	Keywords []string `json:"keywords,omitempty"`

	// Links are a list of descriptive URLs intended to be used to surface additional documentation, dashboards, etc.
	Links []shared.Link `json:"links,omitempty"`

	// Notes contain a human readable snippets intended as a quick start for the users of the Application.
	// CommonMark markdown syntax may be used for rich text representation.
	Notes string `json:"notes,omitempty"`
}

type PackageMeta struct {
	PackageDescriptor `json:",inline"`

	ChartSourceRef `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PackageView struct {
	metav1.TypeMeta `json:",inline"`
	PackageMeta     `json:",inline"`

	// Chart value files
	ValuesFiles []ValuesFile `json:"valuesFiles,omitempty"`

	// openAPIV3Schema describes the schema used for validation and pruning of the Values file.
	// +optional
	OpenAPIV3Schema *crdv1.JSONSchemaProps `json:"openAPIV3Schema,omitempty"`
}

type ValuesFile struct {
	Filename string                `json:"filename"`
	Values   *runtime.RawExtension `json:"values,omitempty"`
}
