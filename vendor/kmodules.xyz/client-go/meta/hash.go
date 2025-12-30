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
	"hash"
	"reflect"
	"strconv"

	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/structs"
	"github.com/zeebo/xxh3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func ResourceHash(obj metav1.Object) string {
	h := xxh3.New()
	_, _ = h.WriteString(string(obj.GetUID()))
	_, _ = h.WriteString(",")
	if obj.GetGeneration() > 0 {
		_, _ = h.WriteString(strconv.FormatInt(obj.GetGeneration(), 10))
	} else {
		_, _ = h.WriteString(ObjectHash(obj))
	}
	return strconv.FormatUint(h.Sum64(), 10)
}

// ObjectHash includes all top label fields (like data, spec) except TypeMeta, ObjectMeta and Status
// also includes Generation, Annotation and Labels form ObjectMeta
func ObjectHash(in metav1.Object) string {
	obj := make(map[string]any)

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

	u, isUnstructured := in.(*unstructured.Unstructured)
	if isUnstructured {
		for fieldName, v := range u.UnstructuredContent() {
			if fieldName != "metadata" && fieldName != "apiVersion" && fieldName != "kind" && fieldName != "status" {
				obj[fieldName] = v
			}
		}
	} else {
		st := structs.New(in)
		for _, field := range st.Fields() {
			fieldName := field.Name()
			if fieldName != "ObjectMeta" && fieldName != "TypeMeta" && fieldName != "Status" {
				obj[fieldName] = field.Value()
			}
		}
	}

	h := xxh3.New()
	DeepHashObject(h, obj)
	return strconv.FormatUint(h.Sum64(), 10)
}

func GenerationHash(in metav1.Object) string {
	obj := make(map[string]any, 3)
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
	h := xxh3.New()
	DeepHashObject(h, obj)
	return strconv.FormatUint(h.Sum64(), 10)
}

// DeepHashObject writes specified object to hash using the spew library
// which follows pointers and prints actual values of the nested objects
// ensuring the hash does not change when a pointer changes.
func DeepHashObject(hasher hash.Hash, objectToWrite any) {
	hasher.Reset()
	printer := spew.ConfigState{
		Indent:         " ",
		SortKeys:       true,
		DisableMethods: true,
		SpewKeys:       true,
	}
	_, _ = printer.Fprintf(hasher, "%#v", objectToWrite)
}

func MustAlreadyReconciled(o any) bool {
	reconciled, err := AlreadyReconciled(o)
	if err != nil {
		panic("failed to extract status.observedGeneration field due to err:" + err.Error())
	}
	return reconciled
}

func AlreadyReconciled(o any) (bool, error) {
	var generation, observedGeneration int64
	var err error

	switch obj := o.(type) {
	case *unstructured.Unstructured:
		generation, observedGeneration, err = extractGenerationFromUnstructured(obj)
	case metav1.Object:
		generation, observedGeneration, err = extractGenerationFromObject(obj)
	default:
		err = fmt.Errorf("unknown object type %s", reflect.TypeOf(o).String())
	}
	if err != nil {
		return false, err
	}
	return observedGeneration == generation, nil
}

func extractGenerationFromUnstructured(obj *unstructured.Unstructured) (int64, int64, error) {
	val, found, err := unstructured.NestedFieldNoCopy(obj.Object, "status", "observedGeneration")
	if err != nil {
		return -1, -1, err
	} else if !found {
		return obj.GetGeneration(), -1, nil
	}
	observedGeneration, ok := val.(int64)
	if !ok {
		return -1, -1, fmt.Errorf("%s %s/%s status.observedGeneration %+v is not int64", obj.GroupVersionKind(), obj.GetNamespace(), obj.GetName(), val)
	}
	return obj.GetGeneration(), observedGeneration, nil
}

func extractGenerationFromObject(obj metav1.Object) (int64, int64, error) {
	st := structs.New(obj)
	fieldStatus, found := st.FieldOk("Status")
	if !found {
		return obj.GetGeneration(), -1, nil
	}
	fieldObsGen, found := fieldStatus.FieldOk("ObservedGeneration")
	if !found {
		return obj.GetGeneration(), -1, nil
	}
	observedGeneration, ok := fieldObsGen.Value().(int64)
	if !ok {
		return -1, -1, fmt.Errorf("%s %s/%s status.observedGeneration %+v is not int64", reflect.TypeOf(obj).String(), obj.GetNamespace(), obj.GetName(), fieldObsGen.Value())
	}
	return obj.GetGeneration(), observedGeneration, nil
}
