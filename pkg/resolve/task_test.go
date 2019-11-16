/*
Copyright The Stash Authors.

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

package resolve

import (
	"testing"

	"stash.appscode.dev/stash/apis/stash/v1beta1"

	"gomodules.xyz/envsubst"
)

func TestResolveWithInputs(t *testing.T) {
	function := v1beta1.Function{
		Spec: v1beta1.FunctionSpec{
			Args: []string{
				"arg",
				"--p1=${p1}",
				"--p2=${p2:=d2}",
				"--p3=${p3:=}",
			},
		},
	}
	inputs := map[string]string{
		"p1": "aa",
	}
	err := resolveWithInputs(&function, inputs)
	if err != nil {
		t.Error(err)
	}
	t.Log(function)

	function = v1beta1.Function{
		Spec: v1beta1.FunctionSpec{
			Args: []string{
				"arg",
				"--p1=${p1}",
				"--p2=${p2}",
			},
		},
	}
	err = resolveWithInputs(&function, inputs)
	if err == nil || !envsubst.IsValueNotFoundError(err) {
		t.Error("Expected ValueNotFoundError")
	}
}
