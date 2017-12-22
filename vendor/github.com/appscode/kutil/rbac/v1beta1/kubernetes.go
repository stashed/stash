package v1beta1

import (
	"errors"

	"github.com/appscode/kutil/meta"
	rbac "k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	return rbac.SchemeGroupVersion.WithKind(meta.GetKind(v))
}

func AssignTypeKind(v interface{}) error {
	_, err := conversion.EnforcePtr(v)
	if err != nil {
		return err
	}

	switch u := v.(type) {
	case *rbac.Role:
		u.APIVersion = rbac.SchemeGroupVersion.String()
		u.Kind = meta.GetKind(v)
		return nil
	case *rbac.RoleBinding:
		u.APIVersion = rbac.SchemeGroupVersion.String()
		u.Kind = meta.GetKind(v)
		return nil
	case *rbac.ClusterRole:
		u.APIVersion = rbac.SchemeGroupVersion.String()
		u.Kind = meta.GetKind(v)
		return nil
	case *rbac.ClusterRoleBinding:
		u.APIVersion = rbac.SchemeGroupVersion.String()
		u.Kind = meta.GetKind(v)
		return nil
	}
	return errors.New("unknown api object type")
}
