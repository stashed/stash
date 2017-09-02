package v1beta1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime/schema"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	switch v.(type) {
	case *apps.StatefulSet:
		return apps.SchemeGroupVersion.WithKind("StatefulSet")
	case *apps.Deployment:
		return apps.SchemeGroupVersion.WithKind("Deployment")
	default:
		return schema.GroupVersionKind{}
	}
}

func AssignTypeKind(v interface{}) error {
	switch u := v.(type) {
	case *apps.StatefulSet:
		u.APIVersion = apps.SchemeGroupVersion.String()
		u.Kind = "StatefulSet"
		return nil
	case *apps.Deployment:
		u.APIVersion = apps.SchemeGroupVersion.String()
		u.Kind = "Deployment"
		return nil
	}
	return errors.New("Unknown api object type")
}
