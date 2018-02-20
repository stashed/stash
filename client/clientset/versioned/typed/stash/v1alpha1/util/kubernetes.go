package util

import (
	"fmt"
	"reflect"

	"github.com/appscode/kutil/meta"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

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
