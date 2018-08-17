package v1

import (
	"fmt"

	"github.com/appscode/go/types"
	"github.com/appscode/mergo"
	"github.com/json-iterator/go"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var json = jsoniter.ConfigFastest

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

func EnsureContainerDeleted(containers []core.Container, name string) []core.Container {
	for i, c := range containers {
		if c.Name == name {
			return append(containers[:i], containers[i+1:]...)
		}
	}
	return containers
}

func UpsertContainer(containers []core.Container, upsert core.Container) []core.Container {
	for i, container := range containers {
		if container.Name == upsert.Name {
			err := mergo.MergeWithOverwrite(&container, upsert)
			if err != nil {
				panic(err)
			}
			containers[i] = container
			return containers
		}
	}
	return append(containers, upsert)
}

func UpsertContainers(containers []core.Container, addons []core.Container) []core.Container {
	var out = containers
	for _, c := range addons {
		out = UpsertContainer(out, c)
	}
	return out
}

func UpsertVolume(volumes []core.Volume, nv ...core.Volume) []core.Volume {
	upsert := func(v core.Volume) {
		for i, vol := range volumes {
			if vol.Name == v.Name {
				volumes[i] = v
				return
			}
		}
		volumes = append(volumes, v)
	}

	for _, volume := range nv {
		upsert(volume)
	}
	return volumes

}

func UpsertVolumeClaim(volumeClaims []core.PersistentVolumeClaim, upsert core.PersistentVolumeClaim) []core.PersistentVolumeClaim {
	for i, vc := range volumeClaims {
		if vc.Name == upsert.Name {
			volumeClaims[i] = upsert
			return volumeClaims
		}
	}
	return append(volumeClaims, upsert)
}

func EnsureVolumeDeleted(volumes []core.Volume, name string) []core.Volume {
	for i, v := range volumes {
		if v.Name == name {
			return append(volumes[:i], volumes[i+1:]...)
		}
	}
	return volumes
}

func UpsertVolumeMount(mounts []core.VolumeMount, nv ...core.VolumeMount) []core.VolumeMount {
	upsert := func(m core.VolumeMount) {
		for i, vol := range mounts {
			if vol.Name == m.Name {
				mounts[i] = m
				return
			}
		}
		mounts = append(mounts, m)
	}

	for _, mount := range nv {
		upsert(mount)
	}
	return mounts
}

func EnsureVolumeMountDeleted(mounts []core.VolumeMount, name string) []core.VolumeMount {
	for i, v := range mounts {
		if v.Name == name {
			return append(mounts[:i], mounts[i+1:]...)
		}
	}
	return mounts
}

func UpsertVolumeMountByPath(mounts []core.VolumeMount, nv core.VolumeMount) []core.VolumeMount {
	for i, vol := range mounts {
		if vol.MountPath == nv.MountPath {
			mounts[i] = nv
			return mounts
		}
	}
	return append(mounts, nv)
}

func EnsureVolumeMountDeletedByPath(mounts []core.VolumeMount, mountPath string) []core.VolumeMount {
	for i, v := range mounts {
		if v.MountPath == mountPath {
			return append(mounts[:i], mounts[i+1:]...)
		}
	}
	return mounts
}

func UpsertEnvVars(vars []core.EnvVar, nv ...core.EnvVar) []core.EnvVar {
	upsert := func(env core.EnvVar) {
		for i, v := range vars {
			if v.Name == env.Name {
				vars[i] = env
				return
			}
		}
		vars = append(vars, env)
	}

	for _, env := range nv {
		upsert(env)
	}
	return vars
}

func EnsureEnvVarDeleted(vars []core.EnvVar, name string) []core.EnvVar {
	for i, v := range vars {
		if v.Name == name {
			return append(vars[:i], vars[i+1:]...)
		}
	}
	return vars
}

func UpsertMap(maps, upsert map[string]string) map[string]string {
	if maps == nil {
		maps = make(map[string]string)
	}
	for k, v := range upsert {
		maps[k] = v
	}
	return maps
}

func MergeLocalObjectReferences(l1, l2 []core.LocalObjectReference) []core.LocalObjectReference {
	result := make([]core.LocalObjectReference, 0, len(l1)+len(l2))
	m := make(map[string]core.LocalObjectReference)
	for _, ref := range l1 {
		m[ref.Name] = ref
		result = append(result, ref)
	}
	for _, ref := range l2 {
		if _, found := m[ref.Name]; !found {
			result = append(result, ref)
		}
	}
	return result
}

func EnsureOwnerReference(meta metav1.ObjectMeta, owner *core.ObjectReference) metav1.ObjectMeta {
	if owner == nil ||
		owner.APIVersion == "" ||
		owner.Kind == "" ||
		owner.Name == "" ||
		owner.UID == "" {
		return meta
	}
	if meta.Namespace != owner.Namespace {
		panic(fmt.Errorf("owner %s %s must be from the same namespace as object %s", owner.Kind, owner.Name, meta.Name))
	}

	fi := -1
	for i, ref := range meta.OwnerReferences {
		if ref.Kind == owner.Kind && ref.Name == owner.Name {
			fi = i
			break
		}
	}
	if fi == -1 {
		meta.OwnerReferences = append(meta.OwnerReferences, metav1.OwnerReference{})
		fi = len(meta.OwnerReferences) - 1
	}
	meta.OwnerReferences[fi].APIVersion = owner.APIVersion
	meta.OwnerReferences[fi].Kind = owner.Kind
	meta.OwnerReferences[fi].Name = owner.Name
	meta.OwnerReferences[fi].UID = owner.UID
	if meta.OwnerReferences[fi].BlockOwnerDeletion == nil {
		meta.OwnerReferences[fi].BlockOwnerDeletion = types.FalseP()
	}
	return meta
}

func RemoveOwnerReference(meta metav1.ObjectMeta, owner *core.ObjectReference) metav1.ObjectMeta {
	for i, ref := range meta.OwnerReferences {
		if ref.Kind == owner.Kind && ref.Name == owner.Name {
			meta.OwnerReferences = append(meta.OwnerReferences[:i], meta.OwnerReferences[i+1:]...)
			break
		}
	}
	return meta
}
