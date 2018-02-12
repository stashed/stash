package meta

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

func CreateStrategicPatch(cur runtime.Object, mod runtime.Object) ([]byte, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, err
	}

	return strategicpatch.CreateTwoWayMergePatch(curJson, modJson, mod)
}

func CreateJSONMergePatch(cur runtime.Object, mod runtime.Object) ([]byte, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, err
	}

	return jsonmergepatch.CreateThreeWayJSONMergePatch(curJson, modJson, curJson)
}
