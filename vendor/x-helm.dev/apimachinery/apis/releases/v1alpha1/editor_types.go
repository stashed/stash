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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	kmapi "kmodules.xyz/client-go/api/v1"
)

type Metadata struct {
	Resource kmapi.ResourceID `json:"resource"`
	Release  ObjectMeta       `json:"release"`
}

type MetadataFlat struct {
	kmapi.ResourceID `json:",inline"`
	ReleaseName      string `json:"releaseName"`
	Namespace        string `json:"namespace"`
}

type ObjectMeta struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ModelMetadata struct {
	Metadata `json:"metadata,omitempty"`
}

type Model struct {
	Metadata  `json:"metadata,omitempty"`
	Resources *unstructured.Unstructured `json:"resources,omitempty"`
}

type ChartOrder struct {
	ChartSourceFlatRef `json:",inline"`

	ReleaseName string                     `json:"releaseName,omitempty"`
	Namespace   string                     `json:"namespace,omitempty"`
	Values      *unstructured.Unstructured `json:"values,omitempty"`
}

type EditorParameters struct {
	ValuesFile string `json:"valuesFile,omitempty"`
	// RFC 6902 compatible json patch. ref: http://jsonpatch.com
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	ValuesPatch *runtime.RawExtension `json:"valuesPatch,omitempty"`
}

type EditResourceOrder struct {
	Group    string `json:"group,omitempty"`
	Version  string `json:"version,omitempty"`
	Resource string `json:"resource,omitempty"`

	ReleaseName string `json:"releaseName,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Values      string `json:"values,omitempty"`
}

type BucketFile struct {
	// URL of the file in bucket
	URL string `json:"url"`
	// Bucket key for this file
	Key      string `json:"key"`
	Filename string `json:"filename"`
	Data     []byte `json:"data"`
}

type BucketObject struct {
	// URL of the file in bucket
	URL string `json:"url,omitempty"`
	// Bucket key for this file
	Key            string `json:"key,omitempty"`
	ResourceObject `json:",inline"`
}

type BucketFileRef struct {
	// URL of the file in bucket
	URL string `json:"url"`
	// Bucket key for this file
	Key string `json:"key"`
}

type ChartTemplate struct {
	ChartSourceRef `json:",inline"`
	ReleaseName    string           `json:"releaseName,omitempty"`
	Namespace      string           `json:"namespace,omitempty"`
	CRDs           []BucketObject   `json:"crds,omitempty"`
	Manifest       *BucketFileRef   `json:"manifest,omitempty"`
	Resources      []ResourceObject `json:"resources,omitempty"`
}

type BucketFileOutput struct {
	// URL of the file in bucket
	URL string `json:"url,omitempty"`
	// Bucket key for this file
	Key      string `json:"key,omitempty"`
	Filename string `json:"filename,omitempty"`
	Data     string `json:"data,omitempty"`
}

type ChartTemplateOutput struct {
	ChartSourceRef `json:",inline"`
	ReleaseName    string             `json:"releaseName,omitempty"`
	Namespace      string             `json:"namespace,omitempty"`
	CRDs           []BucketFileOutput `json:"crds,omitempty"`
	Manifest       *BucketFileRef     `json:"manifest,omitempty"`
	Resources      []ResourceFile     `json:"resources,omitempty"`
}

type EditorTemplate struct {
	Manifest  []byte                     `json:"manifest,omitempty"`
	Values    *unstructured.Unstructured `json:"values,omitempty"`
	Resources []ResourceObject           `json:"resources,omitempty"`
}

type ResourceOutput struct {
	CRDs      []ResourceFile `json:"crds,omitempty"`
	Resources []ResourceFile `json:"resources,omitempty"`
}

type ObjectModel struct {
	Key    string                     `json:"key"`
	Object *unstructured.Unstructured `json:"object"`
}

type ResourceObject struct {
	Filename string                     `json:"filename,omitempty"`
	Key      string                     `json:"key,omitempty"`
	Data     *unstructured.Unstructured `json:"data,omitempty"`
}

type ResourceFile struct {
	Filename string `json:"filename,omitempty"`
	Key      string `json:"key,omitempty"`
	Data     string `json:"data,omitempty"`
}

type SimpleValue struct {
	metav1.TypeMeta `json:",inline,omitempty"`
	ObjectMeta      ObjectMeta `json:"metadata,omitempty"`
}
