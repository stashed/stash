package meta

import (
	jp "github.com/appscode/jsonpatch"
	"github.com/evanphx/json-patch"
	"github.com/json-iterator/go"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

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

	return jsonpatch.CreateMergePatch(curJson, modJson)
}

func CreateJSONPatch(cur runtime.Object, mod runtime.Object) ([]byte, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, err
	}

	ops, err := jp.CreatePatch(curJson, modJson)
	if err != nil {
		return nil, err
	}
	return json.Marshal(ops)
}
