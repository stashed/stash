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

package v1

import (
	core "k8s.io/api/core/v1"
)

type Lineage struct {
	Chain      []ObjectInfo `json:"chain,omitempty" protobuf:"bytes,1,rep,name=chain"`
	Containers []string     `json:"containers,omitempty" protobuf:"bytes,2,rep,name=containers"`
}

type ImageInfo struct {
	Image       string       `json:"image,omitempty" protobuf:"bytes,1,opt,name=image"`
	Lineages    []Lineage    `json:"lineages,omitempty" protobuf:"bytes,2,rep,name=lineages"`
	PullSecrets *PullSecrets `json:"pullSecrets" protobuf:"bytes,3,opt,name=pullSecrets"`
}

type PullSecrets struct {
	Namespace string                      `json:"namespace,omitempty" protobuf:"bytes,1,opt,name=namespace"`
	Refs      []core.LocalObjectReference `json:"refs,omitempty" protobuf:"bytes,2,rep,name=refs"`
}
