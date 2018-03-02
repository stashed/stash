package v1beta1

import (
	"github.com/appscode/kutil/meta"
	"github.com/json-iterator/go"
	"github.com/pkg/errors"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	return extensions.SchemeGroupVersion.WithKind(meta.GetKind(v))
}

func AssignTypeKind(v interface{}) error {
	_, err := conversion.EnforcePtr(v)
	if err != nil {
		return err
	}

	switch u := v.(type) {
	case *extensions.Ingress:
		u.APIVersion = extensions.SchemeGroupVersion.String()
		u.Kind = meta.GetKind(v)
		return nil
	case *extensions.DaemonSet:
		u.APIVersion = extensions.SchemeGroupVersion.String()
		u.Kind = meta.GetKind(v)
		return nil
	case *extensions.ReplicaSet:
		u.APIVersion = extensions.SchemeGroupVersion.String()
		u.Kind = meta.GetKind(v)
		return nil
	case *extensions.Deployment:
		u.APIVersion = extensions.SchemeGroupVersion.String()
		u.Kind = meta.GetKind(v)
		return nil
	}
	return errors.New("unknown api object type")
}

func IsOwnedByDeployment(rs *extensions.ReplicaSet) bool {
	for _, ref := range rs.OwnerReferences {
		if ref.Kind == "Deployment" && ref.Name != "" {
			return true
		}
	}
	return false
}
