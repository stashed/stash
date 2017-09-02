package v1beta1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime/schema"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	switch v.(type) {
	case *extensions.Ingress:
		return extensions.SchemeGroupVersion.WithKind("Ingress")
	case *extensions.DaemonSet:
		return extensions.SchemeGroupVersion.WithKind("DaemonSet")
	case *extensions.ReplicaSet:
		return extensions.SchemeGroupVersion.WithKind("ReplicaSet")
	case *extensions.Deployment:
		return extensions.SchemeGroupVersion.WithKind("Deployment")
	case *extensions.ThirdPartyResource:
		return extensions.SchemeGroupVersion.WithKind("ThirdPartyResource")
	default:
		return schema.GroupVersionKind{}
	}
}

func AssignTypeKind(v interface{}) error {
	switch u := v.(type) {
	case *extensions.Ingress:
		u.APIVersion = extensions.SchemeGroupVersion.String()
		u.Kind = "Ingress"
		return nil
	case *extensions.DaemonSet:
		u.APIVersion = extensions.SchemeGroupVersion.String()
		u.Kind = "DaemonSet"
		return nil
	case *extensions.ReplicaSet:
		u.APIVersion = extensions.SchemeGroupVersion.String()
		u.Kind = "ReplicaSet"
		return nil
	case *extensions.Deployment:
		u.APIVersion = extensions.SchemeGroupVersion.String()
		u.Kind = "Deployment"
		return nil
	case *extensions.ThirdPartyResource:
		u.APIVersion = extensions.SchemeGroupVersion.String()
		u.Kind = "ThirdPartyResource"
		return nil
	}
	return errors.New("Unknown api object type")
}
