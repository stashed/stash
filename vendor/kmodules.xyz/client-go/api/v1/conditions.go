/*
Copyright 2020 The Kubernetes Authors.

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
)

// ConditionSeverity expresses the severity of a Condition Type failing.
type ConditionSeverity string

const (
	// ConditionSeverityError specifies that a condition with `Status=False` is an error.
	ConditionSeverityError ConditionSeverity = "Error"

	// ConditionSeverityWarning specifies that a condition with `Status=False` is a warning.
	ConditionSeverityWarning ConditionSeverity = "Warning"

	// ConditionSeverityInfo specifies that a condition with `Status=False` is informative.
	ConditionSeverityInfo ConditionSeverity = "Info"

	// ConditionSeverityNone should apply only to util with `Status=True`.
	ConditionSeverityNone ConditionSeverity = ""
)

// ConditionType is a valid value for Condition.Type.
type ConditionType string

const (
	// ReadyCondition defines the Ready condition type that summarizes the operational state of an object.
	ReadyCondition ConditionType = "Ready"
)

// Condition defines an observation of a object operational state.
type Condition struct {
	// Type of condition in CamelCase or in foo.example.com/CamelCase.
	// Many .condition.type values are consistent across resources like Available, but because arbitrary util
	// can be useful (see .node.status.util), the ability to deconflict is important.
	Type ConditionType `json:"type" protobuf:"bytes,4,opt,name=type,casttype=ConditionType"`

	// Status of the condition, one of True, False, Unknown.
	Status metav1.ConditionStatus `json:"status" protobuf:"bytes,5,opt,name=status,casttype=k8s.io/api/core/v1.ConditionStatus"`

	// If set, this represents the .metadata.generation that the condition was set based upon.
	// For instance, if .metadata.generation is currently 12, but the .status.condition[x].observedGeneration is 9, the condition is out of date
	// with respect to the current state of the instance.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,3,opt,name=observedGeneration"`

	// Severity provides an explicit classification of Reason code, so the users or machines can immediately
	// understand the current situation and act accordingly.
	// The Severity field MUST be set only when Status=False.
	// +optional
	Severity ConditionSeverity `json:"severity,omitempty" protobuf:"bytes,6,opt,name=severity,casttype=ConditionSeverity"`

	// Last time the condition transitioned from one status to another.
	// This should be when the underlying condition changed. If that is not known, then using the time when
	// the API field changed is acceptable.
	LastTransitionTime metav1.Time `json:"lastTransitionTime" protobuf:"bytes,7,opt,name=lastTransitionTime"`

	// The reason for the condition's last transition in CamelCase.
	// The specific API may choose whether this field is considered a guaranteed API.
	// This field may not be empty.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,8,opt,name=reason"`

	// A human-readable message indicating details about the transition.
	// This field may be empty.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,9,opt,name=message"`
}

// Conditions provide observations of the operational state of a object.
type Conditions []Condition
