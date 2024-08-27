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
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetScaledObject(opsObj map[string]interface{}) (ScaledObject, error) {
	opsPathMapper, err := LoadOpsPathMapper(opsObj)
	if err != nil {
		return nil, err
	}
	dbObj, err := extractReferencedObject(opsObj, opsPathMapper.GetAppRefPath()...)
	if err != nil {
		return nil, err
	}

	// Merge opsObj scaling information with dbObj
	mapping, err := getMapping(opsObj, opsPathMapper)
	if err != nil {
		return nil, err
	}

	for opsPath, dbPath := range mapping {
		sop := splitPathToSlice(string(opsPath))
		sdp := splitPathToSlice(string(dbPath))

		opsVal, found, err := unstructured.NestedFieldCopy(opsObj, sop...)
		if err != nil {
			return nil, err
		}
		if found {
			if err := unstructured.SetNestedField(dbObj, opsVal, sdp...); err != nil {
				return nil, err
			}
		}
	}

	return dbObj, nil
}

func splitPathToSlice(path string) []string {
	return strings.Split(path, ".")
}

func extractReferencedObject(opsObj map[string]interface{}, refDbPath ...string) (map[string]interface{}, error) {
	if len(refDbPath) == 0 {
		return nil, errors.New("refDbPath is empty")
	}
	dbObj, found, _ := unstructured.NestedMap(opsObj, refDbPath...)
	if !found {
		return nil, errors.New("referenced db object not found")
	}

	return dbObj, nil
}

func getGVK(obj map[string]interface{}) schema.GroupVersionKind {
	var unObj unstructured.Unstructured
	unObj.SetUnstructuredContent(obj)

	return unObj.GroupVersionKind()
}

func getScalingType(opsObj map[string]interface{}) (string, error) {
	tp, found, _ := unstructured.NestedString(opsObj, "spec", "type")
	if !found {
		return "", errors.New("scaling type not found")
	}

	return tp, nil
}

func getMapping(opsObj OpsReqObject, opsMapper OpsPathMapper) (map[OpsReqPath]ReferencedObjPath, error) {
	scalingType, err := getScalingType(opsObj)
	if err != nil {
		return nil, err
	}

	switch scalingType {
	case ScalingTypeHorizontal:
		return opsMapper.HorizontalPathMapping(), nil
	case ScalingTypeVertical:
		return opsMapper.VerticalPathMapping(), nil
	case ScalingTypeVolumeExpansion:
		return opsMapper.VolumeExpansionPathMapping(), nil
	}

	return nil, fmt.Errorf("scaling type `%s` not supported", scalingType)
}
