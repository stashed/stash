package v1beta1

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/appscode/kutil"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	return apps.SchemeGroupVersion.WithKind(kutil.GetKind(v))
}

func AssignTypeKind(v interface{}) error {
	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		return fmt.Errorf("%v must be a pointer", v)
	}

	switch u := v.(type) {
	case *apps.StatefulSet:
		u.APIVersion = apps.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *apps.Deployment:
		u.APIVersion = apps.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	}
	return errors.New("unknown api object type")
}

func ObjectReferenceFor(v interface{}) *apiv1.ObjectReference {
	switch u := v.(type) {
	case apps.StatefulSet:
		return &apiv1.ObjectReference{
			APIVersion:      apps.SchemeGroupVersion.String(),
			Kind:            kutil.GetKind(v),
			Namespace:       u.Namespace,
			Name:            u.Name,
			UID:             u.UID,
			ResourceVersion: u.ResourceVersion,
		}
	case *apps.StatefulSet:
		return &apiv1.ObjectReference{
			APIVersion:      apps.SchemeGroupVersion.String(),
			Kind:            kutil.GetKind(v),
			Namespace:       u.Namespace,
			Name:            u.Name,
			UID:             u.UID,
			ResourceVersion: u.ResourceVersion,
		}
	case apps.Deployment:
		return &apiv1.ObjectReference{
			APIVersion:      apps.SchemeGroupVersion.String(),
			Kind:            kutil.GetKind(v),
			Namespace:       u.Namespace,
			Name:            u.Name,
			UID:             u.UID,
			ResourceVersion: u.ResourceVersion,
		}
	case *apps.Deployment:
		return &apiv1.ObjectReference{
			APIVersion:      apps.SchemeGroupVersion.String(),
			Kind:            kutil.GetKind(v),
			Namespace:       u.Namespace,
			Name:            u.Name,
			UID:             u.UID,
			ResourceVersion: u.ResourceVersion,
		}
	}
	return &apiv1.ObjectReference{}
}
