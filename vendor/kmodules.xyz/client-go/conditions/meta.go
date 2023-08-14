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

package conditions

import (
	"fmt"

	kmapi "kmodules.xyz/client-go/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KEP: https://github.com/kubernetes/enhancements/blob/ced773ab59f0ff080888a912ab99474245623dad/keps/sig-api-machinery/1623-standardize-conditions/README.md

// List of common condition types
const (
	ConditionProgressing = "Progressing"
	ConditionInitialized = "Initialized"
	ConditionReady       = "Ready"
	ConditionAvailable   = "Available"
	ConditionFailed      = "Failed"

	ConditionRequestApproved = "Approved"
	ConditionRequestDenied   = "Denied"
)

func NewCondition(reason string, message string, generation int64, conditionStatus ...bool) kmapi.Condition {
	cs := metav1.ConditionTrue
	if len(conditionStatus) > 0 && !conditionStatus[0] {
		cs = metav1.ConditionFalse
	}

	return kmapi.Condition{
		Type:               kmapi.ConditionType(reason),
		Reason:             reason,
		Message:            message,
		Status:             cs,
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: generation,
	}
}

// HasCondition returns "true" if the desired condition provided in "condType" is present in the condition list.
// Otherwise, it returns "false".
func HasCondition(conditions []kmapi.Condition, condType string) bool {
	for i := range conditions {
		if conditions[i].Type == kmapi.ConditionType(condType) {
			return true
		}
	}
	return false
}

// GetCondition returns a pointer to the desired condition referred by "condType". Otherwise, it returns nil.
func GetCondition(conditions []kmapi.Condition, condType string) (int, *kmapi.Condition) {
	for i := range conditions {
		c := conditions[i]
		if c.Type == kmapi.ConditionType(condType) {
			return i, &c
		}
	}
	return -1, nil
}

// SetCondition add/update the desired condition to the condition list. It does nothing if the condition is already in
// its desired state.
func SetCondition(conditions []kmapi.Condition, newCondition kmapi.Condition) []kmapi.Condition {
	idx, curCond := GetCondition(conditions, string(newCondition.Type))
	// If the current condition is in its desired state, we have nothing to do. Just return the original condition list.
	if curCond != nil &&
		curCond.Status == newCondition.Status &&
		curCond.Reason == newCondition.Reason &&
		curCond.Message == newCondition.Message &&
		curCond.ObservedGeneration == newCondition.ObservedGeneration {
		return conditions
	}
	// The desired conditions is not in the condition list or is not in its desired state.
	// Update it if present in the condition list, or append the new condition if it does not present.
	newCondition.LastTransitionTime = metav1.Now()
	if idx == -1 {
		conditions = append(conditions, newCondition)
	} else if newCondition.ObservedGeneration >= curCond.ObservedGeneration {
		// only update if the new condition is based on observed generation at least as updated as the current condition
		conditions[idx] = newCondition
	}
	return conditions
}

// RemoveCondition remove a condition from the condition list referred by "condType" parameter.
func RemoveCondition(conditions []kmapi.Condition, condType string) []kmapi.Condition {
	idx, _ := GetCondition(conditions, condType)
	if idx == -1 {
		// The desired condition is not present in the condition list. So, nothing to do.
		return conditions
	}
	return append(conditions[:idx], conditions[idx+1:]...)
}

// IsConditionTrue returns "true" if the desired condition is in true state.
// It returns "false" if the desired condition is not in "true" state or is not in the condition list.
func IsConditionTrue(conditions []kmapi.Condition, condType string) bool {
	for i := range conditions {
		if conditions[i].Type == kmapi.ConditionType(condType) && conditions[i].Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

// IsConditionFalse returns "true" if the desired condition is in false state.
// It returns "false" if the desired condition is not in "false" state or is not in the condition list.
func IsConditionFalse(conditions []kmapi.Condition, condType string) bool {
	for i := range conditions {
		if conditions[i].Type == kmapi.ConditionType(condType) && conditions[i].Status == metav1.ConditionFalse {
			return true
		}
	}
	return false
}

// IsConditionUnknown returns "true" if the desired condition is in unknown state.
// It returns "false" if the desired condition is not in "unknown" state or is not in the condition list.
func IsConditionUnknown(conditions []kmapi.Condition, condType string) bool {
	for i := range conditions {
		if conditions[i].Type == kmapi.ConditionType(condType) && conditions[i].Status == metav1.ConditionUnknown {
			return true
		}
	}
	return false
}

// Status defines the set of statuses a resource can have.
// Based on kstatus: https://github.com/kubernetes-sigs/cli-utils/tree/master/pkg/kstatus
// +kubebuilder:validation:Enum=InProgress;Failed;Current;Terminating;NotFound;Unknown
type Status string

const (
	// The set of status conditions which can be assigned to resources.
	InProgressStatus  Status = "InProgress"
	FailedStatus      Status = "Failed"
	CurrentStatus     Status = "Current"
	TerminatingStatus Status = "Terminating"
	NotFoundStatus    Status = "NotFound"
	UnknownStatus     Status = "Unknown"
)

var Statuses = []Status{InProgressStatus, FailedStatus, CurrentStatus, TerminatingStatus, UnknownStatus}

// String returns the status as a string.
func (s Status) String() string {
	return string(s)
}

// StatusFromStringOrDie turns a string into a Status. Will panic if the provided string is
// not a valid status.
func StatusFromStringOrDie(text string) Status {
	s := Status(text)
	for _, r := range Statuses {
		if s == r {
			return s
		}
	}
	panic(fmt.Errorf("string has invalid status: %s", s))
}
