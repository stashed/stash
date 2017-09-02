package v1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime/schema"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	switch v.(type) {
	case *apiv1.Pod:
		return apiv1.SchemeGroupVersion.WithKind("Pod")
	case *apiv1.ReplicationController:
		return apiv1.SchemeGroupVersion.WithKind("ReplicationController")
	case *apiv1.ConfigMap:
		return apiv1.SchemeGroupVersion.WithKind("ConfigMap")
	case *apiv1.Secret:
		return apiv1.SchemeGroupVersion.WithKind("Secret")
	case *apiv1.Service:
		return apiv1.SchemeGroupVersion.WithKind("Service")
	case *apiv1.PersistentVolumeClaim:
		return apiv1.SchemeGroupVersion.WithKind("PersistentVolumeClaim")
	case *apiv1.PersistentVolume:
		return apiv1.SchemeGroupVersion.WithKind("PersistentVolume")
	case *apiv1.Node:
		return apiv1.SchemeGroupVersion.WithKind("Node")
	case *apiv1.ServiceAccount:
		return apiv1.SchemeGroupVersion.WithKind("ServiceAccount")
	case *apiv1.Namespace:
		return apiv1.SchemeGroupVersion.WithKind("Namespace")
	case *apiv1.Endpoints:
		return apiv1.SchemeGroupVersion.WithKind("Endpoints")
	case *apiv1.ComponentStatus:
		return apiv1.SchemeGroupVersion.WithKind("ComponentStatus")
	case *apiv1.LimitRange:
		return apiv1.SchemeGroupVersion.WithKind("LimitRange")
	case *apiv1.Event:
		return apiv1.SchemeGroupVersion.WithKind("Event")
	default:
		return schema.GroupVersionKind{}
	}
}

func AssignTypeKind(v interface{}) error {
	switch u := v.(type) {
	case *apiv1.Pod:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = "Pod"
		return nil
	case *apiv1.ReplicationController:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = "ReplicationController"
		return nil
	case *apiv1.ConfigMap:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = "ConfigMap"
		return nil
	case *apiv1.Secret:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = "Secret"
		return nil
	case *apiv1.Service:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = "Service"
		return nil
	case *apiv1.PersistentVolumeClaim:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = "PersistentVolumeClaim"
		return nil
	case *apiv1.PersistentVolume:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = "PersistentVolume"
		return nil
	case *apiv1.Node:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = "Node"
		return nil
	case *apiv1.ServiceAccount:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = "ServiceAccount"
		return nil
	case *apiv1.Namespace:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = "Namespace"
		return nil
	case *apiv1.Endpoints:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = "Endpoints"
		return nil
	case *apiv1.ComponentStatus:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = "ComponentStatus"
		return nil
	case *apiv1.LimitRange:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = "LimitRange"
		return nil
	case *apiv1.Event:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = "Event"
		return nil
	}
	return errors.New("Unknown api object type")
}
