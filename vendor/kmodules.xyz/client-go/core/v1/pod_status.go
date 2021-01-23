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

const (
	PodConditionTypeReady = core.PodConditionType("kubedb.com/Ready")
)

// HasCondition returns "true" if the desired condition provided in "condType" is present in the condition list.
// Otherwise, it returns "false".
func HasPodCondition(conditions []core.PodCondition, condType core.PodConditionType) bool {
	for i := range conditions {
		if conditions[i].Type == condType {
			return true
		}
	}
	return false
}

// GetPodCondition returns a pointer to the desired condition referred by "condType". Otherwise, it returns nil.
func GetPodCondition(conditions []core.PodCondition, condType core.PodConditionType) (int, *core.PodCondition) {
	for i := range conditions {
		c := conditions[i]
		if c.Type == condType {
			return i, &c
		}
	}
	return -1, nil
}

// SetPodCondition add/update the desired condition to the condition list. It does nothing if the condition is already in
// its desired state.
func SetPodCondition(conditions []core.PodCondition, newCondition core.PodCondition) []core.PodCondition {
	idx, curCond := GetPodCondition(conditions, newCondition.Type)
	// The desired conditions is not in the condition list or is not in its desired state.
	// If the current condition status is in its desired state, we have nothing to do. Just return the original condition list.
	// Update it if present in the condition list, or append the new condition if it does not present.
	if curCond == nil || idx == -1 {
		return append(conditions, newCondition)
	} else if curCond.Status == newCondition.Status {
		return conditions
	} else if curCond.Status != newCondition.Status {
		conditions[idx].Status = newCondition.Status
		conditions[idx].LastTransitionTime = newCondition.LastTransitionTime
		conditions[idx].Reason = newCondition.Reason
		conditions[idx].Message = newCondition.Message
	}
	return conditions
}

// RemovePodCondition remove a condition from the condition list referred by "condType" parameter.
func RemovePodCondition(conditions []core.PodCondition, condType core.PodConditionType) []core.PodCondition {
	idx, _ := GetPodCondition(conditions, condType)
	if idx == -1 {
		// The desired condition is not present in the condition list. So, nothing to do.
		return conditions
	}
	return append(conditions[:idx], conditions[idx+1:]...)
}

// IsPodConditionTrue returns "true" if the desired condition is in true state.
// It returns "false" if the desired condition is not in "true" state or is not in the condition list.
func IsPodConditionTrue(conditions []core.PodCondition, condType core.PodConditionType) bool {
	for i := range conditions {
		if conditions[i].Type == condType && conditions[i].Status == core.ConditionTrue {
			return true
		}
	}
	return false
}

// IsPodConditionFalse returns "true" if the desired condition is in false state.
// It returns "false" if the desired condition is not in "false" state or is not in the condition list.
func IsPodConditionFalse(conditions []core.PodCondition, condType core.PodConditionType) bool {
	for i := range conditions {
		if conditions[i].Type == condType && conditions[i].Status == core.ConditionFalse {
			return true
		}
	}
	return false
}

func UpsertPodReadinessGateConditionType(readinessGates []core.PodReadinessGate, conditionType core.PodConditionType) []core.PodReadinessGate {
	for i := range readinessGates {
		if readinessGates[i].ConditionType == conditionType {
			return readinessGates
		}
	}
	return append(readinessGates, core.PodReadinessGate{
		ConditionType: conditionType,
	})
}
