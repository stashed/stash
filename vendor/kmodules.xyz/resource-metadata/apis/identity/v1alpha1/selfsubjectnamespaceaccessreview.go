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
	authorization "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceKindSelfSubjectNamespaceAccessReview = "SelfSubjectNamespaceAccessReview"
	ResourceSelfSubjectNamespaceAccessReview     = "selfsubjectnamespaceaccessreview"
	ResourceSelfSubjectNamespaceAccessReviews    = "selfsubjectnamespaceaccessreviews"
)

// SelfSubjectNamespaceAccessReview checks whether or the current user can perform an action.  Not filling in a
// spec.namespace means "in all namespaces".  Self is a special case, because users should always be able
// to check whether they can perform an action

// +genclient
// +genclient:nonNamespaced
// +genclient:onlyVerbs=create
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
type SelfSubjectNamespaceAccessReview struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec holds information about the request being evaluated.  user and groups must be empty
	Spec SelfSubjectNamespaceAccessReviewSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`

	// Status is filled in by the server and indicates whether the request is allowed or not
	// +optional
	Status SubjectAccessNamespaceReviewStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// SelfSubjectNamespaceAccessReviewSpec is a description of the access request.  Exactly one of ResourceAuthorizationAttributes
// and NonResourceAuthorizationAttributes must be set
type SelfSubjectNamespaceAccessReviewSpec struct {
	// ResourceAuthorizationAttributes describes information for a resource access request
	// +optional
	ResourceAttributes []authorization.ResourceAttributes `json:"resourceAttributes,omitempty"`
	// NonResourceAttributes describes information for a non-resource access request
	// +optional
	NonResourceAttributes []authorization.NonResourceAttributes `json:"nonResourceAttributes,omitempty"`
}

type SubjectAccessNamespaceReviewStatus struct {
	Namespaces []string            `json:"namespaces,omitempty"`
	Projects   map[string][]string `json:"projects,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

type SelfSubjectNamespaceAccessReviewList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SelfSubjectNamespaceAccessReview `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&SelfSubjectNamespaceAccessReview{}, &SelfSubjectNamespaceAccessReviewList{})
}
