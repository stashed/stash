package resolve

import (
	"testing"

	"github.com/appscode/stash/apis/stash/v1beta1"
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
