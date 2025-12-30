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
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	jsoniter "github.com/json-iterator/go"
	jp "gomodules.xyz/jsonpatch/v2"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

var json = jsoniter.ConfigFastest

func toJson(v any) ([]byte, error) {
	if u, ok := v.([]byte); ok {
		return u, nil
	}
	return json.Marshal(v)
}

func CreateStrategicPatch(cur any, mod any, fns ...mergepatch.PreconditionFunc) ([]byte, error) {
	curJson, err := toJson(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := toJson(mod)
	if err != nil {
		return nil, err
	}

	return strategicpatch.CreateTwoWayMergePatch(curJson, modJson, mod, fns...)
}

func CreateJSONMergePatch(cur any, mod any, fns ...mergepatch.PreconditionFunc) ([]byte, error) {
	curJson, err := toJson(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := toJson(mod)
	if err != nil {
		return nil, err
	}

	patch, err := jsonpatch.CreateMergePatch(curJson, modJson)
	if err != nil {
		return nil, err
	}
	if err := meetPreconditions(patch, fns...); err != nil {
		return nil, err
	}

	return patch, nil
}

func CreateJSONPatch(cur any, mod any) ([]byte, error) {
	curJson, err := toJson(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := toJson(mod)
	if err != nil {
		return nil, err
	}

	ops, err := jp.CreatePatch(curJson, modJson)
	if err != nil {
		return nil, err
	}
	return json.Marshal(ops)
}

// Apply the preconditions to the patch, and return an error if any of them fail.
// ref: https://github.com/kubernetes/apimachinery/blob/master/pkg/util/jsonmergepatch/patch.go#L74
func meetPreconditions(patch []byte, fns ...mergepatch.PreconditionFunc) error {
	var patchMap map[string]any
	if err := json.Unmarshal(patch, &patchMap); err != nil {
		return fmt.Errorf("failed to unmarshal patch for precondition check: %s", patch)
	}

	for _, fn := range fns {
		if !fn(patchMap) {
			return mergepatch.NewErrPreconditionFailed(patchMap)
		}
	}
	return nil
}

// RequireChainKeyUnchanged creates a precondition function that fails
// if the [field].key is present in the patch (indicating its value
// has changed). Here, [field] can be recursive field i.e. 'spec.someField.someKey'

// Use 'mergepatch' package to set 'RequireKeyUnchanged' and 'RequireMetadataKeyUnchanged'.
// But, for recursive key checking, use the following 'RequireChainKeyUnchanged' method.
// ref: https://github.com/kubernetes/apimachinery/blob/master/pkg/util/mergepatch/util.go#L30

func RequireChainKeyUnchanged(key string) mergepatch.PreconditionFunc {
	return func(patch any) bool {
		patchMap, ok := patch.(map[string]any)
		if !ok {
			fmt.Println("Invalid data")
			return true
		}
		return checkChainKeyUnchanged(key, patchMap)
	}
}

func checkChainKeyUnchanged(key string, mapData map[string]any) bool {
	keys := strings.Split(key, ".")

	newKey := strings.Join(keys[1:], ".")
	if keys[0] == "*" {
		if len(keys) == 1 {
			return true
		}
		for _, val := range mapData {
			if !checkChainKeyUnchanged(newKey, val.(map[string]any)) {
				return false
			}
		}
	} else {
		values, ok := mapData[keys[0]]
		if !ok || len(keys) == 1 {
			return !ok
		}
		if x, ok := values.([]any); ok {
			// x is of type []Interface
			for _, val := range x {
				return checkChainKeyUnchanged(newKey, val.(map[string]any))
			}
		}
		return checkChainKeyUnchanged(newKey, values.(map[string]any))
	}
	return true
}
