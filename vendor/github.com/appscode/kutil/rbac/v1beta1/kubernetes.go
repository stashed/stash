package v1beta1

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/appscode/kutil"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	return rbacv1beta1.SchemeGroupVersion.WithKind(kutil.GetKind(v))
}

func AssignTypeKind(v interface{}) error {
	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		return fmt.Errorf("%v must be a pointer", v)
	}

	switch u := v.(type) {
	case *rbacv1beta1.Role:
		u.APIVersion = rbacv1beta1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *rbacv1beta1.RoleBinding:
		u.APIVersion = rbacv1beta1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *rbacv1beta1.ClusterRole:
		u.APIVersion = rbacv1beta1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *rbacv1beta1.ClusterRoleBinding:
		u.APIVersion = rbacv1beta1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	}
	return errors.New("unknown api object type")
}
