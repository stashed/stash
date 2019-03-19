package resolve

import (
	"fmt"
	"testing"

	"github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestResolveWithInputs(t *testing.T) {
	function := v1beta1.Function{
		Spec: v1beta1.FunctionSpec{
			Args: []string{
				"arg",
				"--p1=${p1}",
				"--p2=${p2}",
				"--p3=${p3}",
				"--p4=${p4=##}",
			},
		},
	}
	inputs := map[string]string{
		"p1": "aa",
		"p2": "bb",
	}
	err := resolveWithInputs(&function, inputs)
	fmt.Println(function.Spec.Args)
	assert.Equal(t, err, nil)

	function.Spec.Args = removeEmptyFlags(function.Spec.Args)
	fmt.Println(function.Spec.Args)
}
