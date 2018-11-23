package meta

import (
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
		diff := Diff(nu, old)
		glog.V(log.LevelDebug).Infof("%s %s/%s has changed. Diff: %s", GetKind(old), oldObj.GetNamespace(), oldObj.GetName(), diff)
	}
	return match
}
