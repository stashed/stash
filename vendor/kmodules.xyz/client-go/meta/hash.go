package meta

import (
	"fmt"
	"hash"
	"hash/fnv"
	"reflect"
	"strconv"

	"github.com/appscode/go/encoding/json/types"
	"github.com/appscode/go/log"
	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/structs"
	"github.com/golang/glog"
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

// Deprecated, should not be used after we drop support for Kubernetes 1.10. Use AlreadyReconciled
func AlreadyObserved(o interface{}, enableStatusSubresource bool) bool {
	if !enableStatusSubresource {
		return false
	}

	obj := o.(metav1.Object)
	st := structs.New(o)

	cur := types.NewIntHash(obj.GetGeneration(), GenerationHash(obj))
	observed, err := types.ParseIntHash(st.Field("Status").Field("ObservedGeneration").Value())
	if err != nil {
		panic(err)
	}
	return observed.Equal(cur)
}

// Deprecated, should not be used after we drop support for Kubernetes 1.10. Use AlreadyReconciled
func AlreadyObserved2(old, nu interface{}, enableStatusSubresource bool) bool {
	if old == nil {
		return nu == nil
	}
	if nu == nil { // && old != nil
		return false
	}
	if old == nu {
		return true
	}

	oldObj := old.(metav1.Object)
	nuObj := nu.(metav1.Object)

	oldStruct := structs.New(old)
	nuStruct := structs.New(nu)

	var match bool

	if enableStatusSubresource {
		observed, err := types.ParseIntHash(nuStruct.Field("Status").Field("ObservedGeneration").Value())
		if err != nil {
			panic(err)
		}
		gen := types.NewIntHash(nuObj.GetGeneration(), GenerationHash(nuObj))
		match = gen.Equal(observed)
	} else {
		match = Equal(oldStruct.Field("Spec").Value(), nuStruct.Field("Spec").Value())
		if match {
			match = reflect.DeepEqual(oldObj.GetLabels(), nuObj.GetLabels())
		}
		if match {
			match = EqualAnnotation(oldObj.GetAnnotations(), nuObj.GetAnnotations())
		}
	}

	if !match && bool(glog.V(log.LevelDebug)) {
		diff := Diff(old, nu)
		glog.V(log.LevelDebug).Infof("%s %s/%s has changed. Diff: %s", GetKind(old), oldObj.GetNamespace(), oldObj.GetName(), diff)
	}
	return match
}
