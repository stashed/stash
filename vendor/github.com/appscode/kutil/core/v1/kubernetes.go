package v1

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/appscode/kutil"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetGroupVersionKind(v interface{}) schema.GroupVersionKind {
	return apiv1.SchemeGroupVersion.WithKind(kutil.GetKind(v))
}

func AssignTypeKind(v interface{}) error {
	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		return fmt.Errorf("%v must be a pointer", v)
	}

	switch u := v.(type) {
	case *apiv1.Pod:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *apiv1.ReplicationController:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *apiv1.ConfigMap:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *apiv1.Secret:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *apiv1.Service:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *apiv1.PersistentVolumeClaim:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *apiv1.PersistentVolume:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *apiv1.Node:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *apiv1.ServiceAccount:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *apiv1.Namespace:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *apiv1.Endpoints:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *apiv1.ComponentStatus:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *apiv1.LimitRange:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	case *apiv1.Event:
		u.APIVersion = apiv1.SchemeGroupVersion.String()
		u.Kind = kutil.GetKind(v)
		return nil
	}
	return errors.New("unknown api object type")
}

func RemoveNextInitializer(m metav1.ObjectMeta) metav1.ObjectMeta {
	if m.GetInitializers() != nil {
		pendingInitializers := m.GetInitializers().Pending
		// Remove self from the list of pending Initializers while preserving ordering.
		if len(pendingInitializers) == 1 {
			m.Initializers = nil
		} else {
			m.Initializers.Pending = append(pendingInitializers[:0], pendingInitializers[1:]...)
		}
	}
	return m
}

func AddFinalizer(m metav1.ObjectMeta, finalizer string) metav1.ObjectMeta {
	for _, name := range m.Finalizers {
		if name == finalizer {
			return m
		}
	}
	m.Finalizers = append(m.Finalizers, finalizer)
	return m
}

func HasFinalizer(m metav1.ObjectMeta, finalizer string) bool {
	for _, name := range m.Finalizers {
		if name == finalizer {
			return true
		}
	}
	return false
}

func RemoveFinalizer(m metav1.ObjectMeta, finalizer string) metav1.ObjectMeta {
	// https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating
	r := m.Finalizers[:0]
	for _, name := range m.Finalizers {
		if name != finalizer {
			r = append(r, name)
		}
	}
	m.Finalizers = r
	return m
}

func EnsureContainerDeleted(containers []apiv1.Container, name string) []apiv1.Container {
	for i, c := range containers {
		if c.Name == name {
			return append(containers[:i], containers[i+1:]...)
		}
	}
	return containers
}

func UpsertContainer(containers []apiv1.Container, nv apiv1.Container) []apiv1.Container {
	for i, vol := range containers {
		if vol.Name == nv.Name {
			containers[i] = nv
			return containers
		}
	}
	return append(containers, nv)
}

func GetString(m map[string]string, key string) string {
	if m == nil {
		return ""
	}
	return m[key]
}

func UpsertVolume(volumes []apiv1.Volume, nv apiv1.Volume) []apiv1.Volume {
	for i, vol := range volumes {
		if vol.Name == nv.Name {
			volumes[i] = nv
			return volumes
		}
	}
	return append(volumes, nv)
}

func EnsureVolumeDeleted(volumes []apiv1.Volume, name string) []apiv1.Volume {
	for i, v := range volumes {
		if v.Name == name {
			return append(volumes[:i], volumes[i+1:]...)
		}
	}
	return volumes
}

func UpsertVolumeMount(mounts []apiv1.VolumeMount, nv apiv1.VolumeMount) []apiv1.VolumeMount {
	for i, vol := range mounts {
		if vol.Name == nv.Name {
			mounts[i] = nv
			return mounts
		}
	}
	return append(mounts, nv)
}

func EnsureVolumeMountDeleted(mounts []apiv1.VolumeMount, name string) []apiv1.VolumeMount {
	for i, v := range mounts {
		if v.Name == name {
			return append(mounts[:i], mounts[i+1:]...)
		}
	}
	return mounts
}

func UpsertEnvVar(vars []apiv1.EnvVar, nv apiv1.EnvVar) []apiv1.EnvVar {
	for i, vol := range vars {
		if vol.Name == nv.Name {
			vars[i] = nv
			return vars
		}
	}
	return append(vars, nv)
}

func EnsureEnvVarDeleted(vars []apiv1.EnvVar, name string) []apiv1.EnvVar {
	for i, v := range vars {
		if v.Name == name {
			return append(vars[:i], vars[i+1:]...)
		}
	}
	return vars
}
