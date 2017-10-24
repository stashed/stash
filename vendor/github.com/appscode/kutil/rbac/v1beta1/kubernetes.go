package v1beta1

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/appscode/kutil"
	rbac "k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	return rbac.SchemeGroupVersion.WithKind(kutil.GetKind(v))
}

func AssignTypeKind(v interface{}) error {
	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		return fmt.Errorf("%v must be a pointer", v)
	}

	switch u := v.(type) {
	case *rbac.Role:
		u.APIVersion = rbac.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *rbac.RoleBinding:
		u.APIVersion = rbac.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *rbac.ClusterRole:
		u.APIVersion = rbac.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *rbac.ClusterRoleBinding:
		u.APIVersion = rbac.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	}
	return errors.New("unknown api object type")
}
