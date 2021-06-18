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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kmapi "kmodules.xyz/client-go/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EventType string

const (
	// ref: https://github.com/cloudevents/spec/blob/v1.0.1/spec.md#type

	EventCreated EventType = "builders.byte.auditor.created.v1"
	EventUpdated EventType = "builders.byte.auditor.updated.v1"
	EventDeleted EventType = "builders.byte.auditor.deleted.v1"
)

// +k8s:deepcopy-gen=false
type Event struct {
	LicenseID   string                                                          `json:"licenseID,omitempty"`
	ResourceID  kmapi.ResourceID                                                `json:"resourceID,omitempty"`
	Resource    client.Object                                                   `json:"resource,omitempty"`
	Connections map[schema.GroupVersionResource][]*metav1.PartialObjectMetadata `json:"connections,omitempty"`
}

type UnstructuredEvent struct {
	LicenseID   string                                                          `json:"licenseID,omitempty"`
	ResourceID  kmapi.ResourceID                                                `json:"resourceID,omitempty"`
	Resource    *unstructured.Unstructured                                      `json:"resource,omitempty"`
	Connections map[schema.GroupVersionResource][]*metav1.PartialObjectMetadata `json:"connections,omitempty"`
}
