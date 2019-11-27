/*
Copyright The Kmodules Authors.

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
	"hash"
	"hash/fnv"
	"reflect"
	"strconv"

	"github.com/appscode/go/encoding/json/types"
	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/structs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ObjectHash includes all top label fields (like data, spec) except TypeMeta, ObjectMeta and Status
// also includes Generation, Annotation and Labels form ObjectMeta
func ObjectHash(in metav1.Object) string {
	obj := make(map[string]interface{})

	obj["generation"] = in.GetGeneration()
	if len(in.GetLabels()) > 0 {
		obj["labels"] = in.GetLabels()
	}

	if len(in.GetAnnotations()) > 0 {
		data := make(map[string]string, len(in.GetAnnotations()))
		for k, v := range in.GetAnnotations() {
			if k != LastAppliedConfigAnnotation {
				data[k] = v
			}
		}
		obj["annotations"] = data
	}

	st := structs.New(in)
	for _, field := range st.Fields() {
		fieldName := field.Name()
		if fieldName != "ObjectMeta" && fieldName != "TypeMeta" && fieldName != "Status" {
			obj[fieldName] = field.Value()
		}
	}

	h := fnv.New64a()
	DeepHashObject(h, obj)
	return strconv.FormatUint(h.Sum64(), 10)
}

func GenerationHash(in metav1.Object) string {
	obj := make(map[string]interface{}, 3)
	obj["generation"] = in.GetGeneration()
	if len(in.GetLabels()) > 0 {
		obj["labels"] = in.GetLabels()
	}
	if len(in.GetAnnotations()) > 0 {
		data := make(map[string]string, len(in.GetAnnotations()))
		for k, v := range in.GetAnnotations() {
			if k != LastAppliedConfigAnnotation {
				data[k] = v
			}
		}
		obj["annotations"] = data
	}
	h := fnv.New64a()
	DeepHashObject(h, obj)
	return strconv.FormatUint(h.Sum64(), 10)
}

// DeepHashObject writes specified object to hash using the spew library
// which follows pointers and prints actual values of the nested objects
// ensuring the hash does not change when a pointer changes.
func DeepHashObject(hasher hash.Hash, objectToWrite interface{}) {
	hasher.Reset()
	printer := spew.ConfigState{
		Indent:         " ",
		SortKeys:       true,
		DisableMethods: true,
		SpewKeys:       true,
	}
	printer.Fprintf(hasher, "%#v", objectToWrite)
}

func AlreadyReconciled(o interface{}) bool {
	var generation, observedGeneration *types.IntHash
	var err error

	switch obj := o.(type) {
	case *unstructured.Unstructured:
		generation = types.IntHashForGeneration(obj.GetGeneration())
		var val interface{}
		val, _, err = unstructured.NestedFieldNoCopy(obj.Object, "status", "observedGeneration")
		if err == nil {
			observedGeneration, err = types.ParseIntHash(val)
		}
	case metav1.Object:
		st := structs.New(o)
		generation = types.IntHashForGeneration(obj.GetGeneration())
		observedGeneration, err = types.ParseIntHash(st.Field("Status").Field("ObservedGeneration").Value())
	default:
		err = fmt.Errorf("unknown object type %s", reflect.TypeOf(o).String())
	}
	if err != nil {
		panic("failed to extract status.observedGeneration field due to err:" + err.Error())
	}
	return observedGeneration.MatchGeneration(generation)
}
