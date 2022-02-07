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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	ResourceKindGenericResourceService = "GenericResourceService"
	ResourceGenericResourceService     = "genericresourceservice"
	ResourceGenericResourceServices    = "genericresourceservices"
)

// GenericResourceService reports any ops services used by any resource

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type GenericResourceService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GenericResourceServiceSpec `json:"spec,omitempty"`
	Status *runtime.RawExtension      `json:"status,omitempty"`
}

type GenericResourceServiceSpec struct {
	Cluster    kmapi.ClusterMetadata            `json:"cluster,omitempty"`
	APIType    kmapi.ResourceID                 `json:"apiType"`
	Name       string                           `json:"name,omitempty"`
	Facilities GenericResourceServiceFacilities `json:"facilities,omitempty"`
	Status     GenericResourceServiceStatus     `json:"status"`
}

type GenericResourceServiceFacilities struct {
	Exposed    GenericResourceServiceFacilitator `json:"exposed,omitempty"`
	TLS        GenericResourceServiceFacilitator `json:"tls,omitempty"`
	Backup     GenericResourceServiceFacilitator `json:"backup,omitempty"`
	Monitoring GenericResourceServiceFacilitator `json:"monitoring,omitempty"`
}

type GenericResourceServiceFacilitator struct {
	Usage FacilityUsage `json:"usage"`
	// +optional
	Resource *kmapi.ResourceID `json:"resource,omitempty"`
	// +optional
	Refs []kmapi.ObjectReference `json:"refs,omitempty"`
}

type FacilityUsage string

const (
	FacilityUsed    FacilityUsage = "Used"
	FacilityUnused  FacilityUsage = "Unused"
	FacilityUnknown FacilityUsage = "Unknown"
)

type GenericResourceServiceStatus struct {
	// Status
	Status string `json:"status,omitempty"`
	// Message
	Message string `json:"message,omitempty"`
}

// GenericResourceServiceList contains a list of GenericResourceService

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type GenericResourceServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GenericResourceService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GenericResourceService{}, &GenericResourceServiceList{})
}
