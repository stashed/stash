package util

import (
	"fmt"
	"reflect"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"kmodules.xyz/client-go/meta"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
)

var json = jsoniter.ConfigFastest

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	return api.SchemeGroupVersion.WithKind(meta.GetKind(v))
}

func AssignTypeKind(v interface{}) error {
	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		return fmt.Errorf("%v must be a pointer", v)
	}

	switch u := v.(type) {
	case *api.Restic:
		u.APIVersion = api.SchemeGroupVersion.String()
		u.Kind = meta.GetKind(v)
		return nil
	}
	return errors.New("unknown api object type")
}
