package v1beta1

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/appscode/kutil"
	"k8s.io/apimachinery/pkg/runtime/schema"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	return extensions.SchemeGroupVersion.WithKind(kutil.GetKind(v))
}

func AssignTypeKind(v interface{}) error {
	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		return fmt.Errorf("%v must be a pointer", v)
	}

	switch u := v.(type) {
	case *extensions.Ingress:
		u.APIVersion = extensions.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *extensions.DaemonSet:
		u.APIVersion = extensions.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *extensions.ReplicaSet:
		u.APIVersion = extensions.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *extensions.Deployment:
		u.APIVersion = extensions.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *extensions.ThirdPartyResource:
		u.APIVersion = extensions.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	}
	return errors.New("unknown api object type")
}
