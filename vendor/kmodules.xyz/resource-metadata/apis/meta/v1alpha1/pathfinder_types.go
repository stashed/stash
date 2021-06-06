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
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	ResourceKindPathFinder = "PathFinder"
	ResourcePathFinder     = "pathfinder"
	ResourcePathFinders    = "pathfinders"
)

// +genclient
// +genclient:nonNamespaced
// +genclient:skipVerbs=get,list,update,patch,delete,deleteCollection,watch,updateStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PathFinder struct {
	metav1.TypeMeta `json:",inline"`
	// Request describes the attributes for the path request.
	// +optional
	Request *PathRequest `json:"request,omitempty"`
	// Response describes the attributes for the path response.
	// +optional
	Response *PathResponse `json:"response,omitempty"`
}

type PathRequest struct {
	Source GroupVersionResource  `json:"source"`
	Target *GroupVersionResource `json:"target,omitempty"`
}

type PathResponse struct {
	Paths []Path `json:"paths,omitempty"`
}

type Path struct {
	Source   GroupVersionResource `json:"source"`
	Target   GroupVersionResource `json:"target"`
	Distance uint64               `json:"distance"`
	Edges    []*Edge              `json:"edges"`
}

type Edge struct {
	Src        GroupVersionResource   `json:"source"`
	Dst        GroupVersionResource   `json:"target"`
	W          uint64                 `json:"distance"`
	Connection ResourceConnectionSpec `json:"connection"`
	Forward    bool                   `json:"forward"`
}

type GroupResources struct {
	Groups   []string `json:"groups"`
	Resource string   `json:"resource"`
}

// GroupVersionResource unambiguously identifies a resource.
type GroupVersionResource struct {
	Group    string `json:"group"`
	Version  string `json:"version"`
	Resource string `json:"resource"`
}

func (in GroupVersionResource) GVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    in.Group,
		Version:  in.Version,
		Resource: in.Resource,
	}
}

func FromGVR(in schema.GroupVersionResource) GroupVersionResource {
	return GroupVersionResource{
		Group:    in.Group,
		Version:  in.Version,
		Resource: in.Resource,
	}
}
