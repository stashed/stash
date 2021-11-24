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

package meta

import (
	"fmt"
	"reflect"

	"github.com/fatih/structs"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
)

type Condition struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	ObservedGeneration int64  `json:"observedGeneration,omitempty"`
}

func StatusEqual(old, new interface{}) bool {
	oldStatus, oldExists := extractStatusFromObject(old)
	newStatus, newExists := extractStatusFromObject(new)
	if oldExists && newExists {
		result := apiequality.Semantic.DeepEqual(oldStatus, newStatus)
		if !result && klog.V(8).Enabled() {
			if diff, err := JsonDiff(oldStatus, newStatus); err == nil {
				klog.V(8).Infoln(diff)
			}
		}
		return result
	}
	return !oldExists && !newExists
}

func StatusConditionAwareEqual(old, new interface{}) bool {
	oldStatus, oldExists := extractStatusFromObject(old)
	newStatus, newExists := extractStatusFromObject(new)
	if oldExists && newExists {
		oldKind := reflect.TypeOf(oldStatus).Kind()
		newKind := reflect.TypeOf(newStatus).Kind()
		if oldKind != newKind {
			klog.Warningf("old status kind %s does not match new status kind %s", oldKind, newKind)
			return false
		}

		var result bool
		if oldKind == reflect.Map {
			result = statusMapEqual(oldStatus.(map[string]interface{}), newStatus.(map[string]interface{}))
		} else {
			oldStruct := structs.New(oldStatus)
			oldStruct.TagName = "json"

			newStruct := structs.New(newStatus)
			newStruct.TagName = "json"

			result = statusMapEqual(oldStruct.Map(), newStruct.Map())
		}
		if !result && klog.V(8).Enabled() {
			if diff, err := JsonDiff(oldStatus, newStatus); err == nil {
				klog.V(8).Infoln(diff)
			}
		}
		return result
	}
	return !oldExists && !newExists
}

func extractStatusFromObject(o interface{}) (interface{}, bool) {
	switch obj := o.(type) {
	case *unstructured.Unstructured:
		v, ok, _ := unstructured.NestedFieldNoCopy(obj.Object, "status")
		return v, ok && v != nil
	case metav1.Object:
		st := structs.New(obj)
		field, ok := st.FieldOk("Status")
		if !ok {
			return nil, ok
		}
		return field.Value(), true
	}
	panic(fmt.Errorf("unknown object %v", reflect.TypeOf(o)))
}

func conditionsEqual(old, nu []Condition) bool {
	// optimization
	if len(old) != len(nu) {
		return false
	}
	oldMap := make(map[Condition]bool, len(old))
	for _, c := range old {
		oldMap[c] = true
	}
	for _, c := range nu {
		if !oldMap[c] {
			return false
		}
	}
	return true
}

func statusMapEqual(old, nu map[string]interface{}) bool {
	// optimization
	if len(old) != len(nu) {
		return false
	}

	for key, oldVal := range old {
		newVal, ok := nu[key]
		if !ok {
			return false
		}
		if key == "conditions" {
			// special case
			oldCond := make([]Condition, 0)
			if err := DecodeObject(oldVal, &oldCond); err != nil {
				klog.Errorln(err)
				return false
			}
			nuCond := make([]Condition, 0)
			if err := DecodeObject(newVal, &nuCond); err != nil {
				klog.Errorln(err)
				return false
			}
			if !conditionsEqual(oldCond, nuCond) {
				return false
			}
		} else if !apiequality.Semantic.DeepEqual(oldVal, newVal) {
			return false
		}
	}

	for key := range nu {
		if _, ok := old[key]; !ok {
			return false
		}
	}
	return true
}
