/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"sort"

	jsoniter "github.com/json-iterator/go"
	"gomodules.xyz/mergo"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var json = jsoniter.ConfigFastest

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

func GetContainerByName(containers []core.Container, name string) *core.Container {
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}
	return nil
}

func UpsertContainer(containers []core.Container, upsert core.Container) []core.Container {
	for i, container := range containers {
		if container.Name == upsert.Name {
			err := mergo.Merge(&container, upsert, mergo.WithOverride)
			if err != nil {
				panic(err)
			}
			// mergo does not overwrite "dst (container)" using empty "src (upsert)" values.
			// This causes problem we want to remove args or commands (eg, disable TLS).
			// TODO: should this be done for all the []string type fields (eg, EnvFrom etc.)?
			container.Command = upsert.Command
			container.Args = upsert.Args
			for i, env := range upsert.Env {
				if env.ValueFrom != nil &&
					env.ValueFrom.FieldRef != nil &&
					env.ValueFrom.FieldRef.APIVersion == "" {
					env.ValueFrom.FieldRef.APIVersion = "v1"
				}
				upsert.Env[i] = env
			}
			container.Env = upsert.Env
			container.VolumeMounts = upsert.VolumeMounts
			container.VolumeDevices = upsert.VolumeDevices
			container.Resources = upsert.Resources
			containers[i] = container
			return containers
		}
	}
	return append(containers, upsert)
}

func UpsertContainers(containers []core.Container, addons []core.Container) []core.Container {
	out := containers
	for _, c := range addons {
		out = UpsertContainer(out, c)
	}
	return out
}

func DeleteContainer(containers []core.Container, name string) []core.Container {
	for i, v := range containers {
		if v.Name == name {
			return append(containers[:i], containers[i+1:]...)
		}
	}
	return containers
}

func GetVolumeByName(volumes []core.Volume, name string) *core.Volume {
	for i := range volumes {
		if volumes[i].Name == name {
			return &volumes[i]
		}
	}
	return nil
}

func UpsertVolume(volumes []core.Volume, nv ...core.Volume) []core.Volume {
	upsert := func(v core.Volume) {
		for i, vol := range volumes {
			if vol.Name == v.Name {
				err := mergo.Merge(&volumes[i], v, mergo.WithOverride)
				if err != nil {
					panic(err)
				}
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

func ReplaceVolumes(existing []core.Volume, desired ...core.Volume) ([]core.Volume, error) {
	merge := func(cur core.Volume) error {
		for i, v := range desired {
			if v.Name == cur.Name {
				if err := mergo.Merge(&cur, v, mergo.WithOverride); err != nil {
					return err
				}
				desired[i] = cur
				break
			}
		}
		return nil
	}

	for _, cur := range existing {
		if err := merge(cur); err != nil {
			return nil, err
		}
	}
	sort.Slice(desired, func(i, j int) bool {
		return desired[i].Name < desired[j].Name
	})
	return desired, nil
}

func MustReplaceVolumes(existing []core.Volume, desired ...core.Volume) []core.Volume {
	result, err := ReplaceVolumes(existing, desired...)
	if err != nil {
		panic(err)
	}
	return result
}

func UpsertVolumeClaim(volumeClaims []core.PersistentVolumeClaim, upsert core.PersistentVolumeClaim) []core.PersistentVolumeClaim {
	for i, vc := range volumeClaims {
		if vc.Name == upsert.Name {
			volumeClaims[i].Labels = upsert.Labels
			volumeClaims[i].Annotations = upsert.Annotations
			if err := mergo.Merge(&volumeClaims[i].Spec, upsert.Spec, mergo.WithOverride); err != nil {
				panic(err)
			}
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

func GetVolumeMountByName(volumeMounts []core.VolumeMount, name string) *core.VolumeMount {
	for i := range volumeMounts {
		if volumeMounts[i].Name == name {
			return &volumeMounts[i]
		}
	}
	return nil
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

func GetEnvByName(envs []core.EnvVar, name string) *core.EnvVar {
	for i := range envs {
		if envs[i].Name == name {
			return &envs[i]
		}
	}
	return nil
}

func UpsertEnvVars(vars []core.EnvVar, nv ...core.EnvVar) []core.EnvVar {
	upsert := func(env core.EnvVar) {
		if env.ValueFrom != nil &&
			env.ValueFrom.FieldRef != nil &&
			env.ValueFrom.FieldRef.APIVersion == "" {
			env.ValueFrom.FieldRef.APIVersion = "v1"
		}

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
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// NewOwnerRef creates an OwnerReference pointing to the given owner.
func NewOwnerRef(owner metav1.Object, gvk schema.GroupVersionKind) *metav1.OwnerReference {
	blockOwnerDeletion := false
	isController := false
	return &metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               owner.GetName(),
		UID:                owner.GetUID(),
		BlockOwnerDeletion: &blockOwnerDeletion,
		Controller:         &isController,
	}
}

// EnsureOwnerReference adds owner if absent or syncs owner if already present.
//
// If you are writing a controller or an operator, use the following code snippet for dependent objects.
// Here, `controller = true` and `blockOwnerDeletion = true`
//
// owner := metav1.NewControllerRef(foo, samplev1alpha1.SchemeGroupVersion.WithKind("Foo"))
// EnsureOwnerReference(dependent, owner)
//
// If our CRD is not a controller but just want to be a owner, use the following code snippet.
// Here, `controller = false` and `blockOwnerDeletion = false`
//
// owner := NewOwnerRef(foo, samplev1alpha1.SchemeGroupVersion.WithKind("Foo"))
// EnsureOwnerReference(dependent, owner)
//
// To understand the impact of `blockOwnerDeletion`, read:
// - https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#foreground-cascading-deletion
// - https://github.com/kubernetes/apimachinery/blob/v0.17.0/pkg/apis/meta/v1/types.go#L297-L323
func EnsureOwnerReference(dependent metav1.Object, owner *metav1.OwnerReference) {
	if owner == nil {
		return
	}

	refs := dependent.GetOwnerReferences()

	fi := -1
	for i := range refs {
		if refs[i].UID == owner.UID {
			fi = i
			break
		}
	}
	if fi == -1 {
		refs = append(refs, *owner)
	} else {
		refs[fi] = *owner
	}

	dependent.SetOwnerReferences(refs)
}

func RemoveOwnerReference(dependent metav1.Object, owner metav1.Object) {
	refs := dependent.GetOwnerReferences()
	for i := range refs {
		if refs[i].UID == owner.GetUID() {
			refs = append(refs[:i], refs[i+1:]...)
			break
		}
	}
	dependent.SetOwnerReferences(refs)
}

// IsOwnedBy checks if the dependent has a owner reference to the given owner
func IsOwnedBy(dependent metav1.Object, owner metav1.Object) (owned bool, controller bool) {
	refs := dependent.GetOwnerReferences()
	for i := range refs {
		if refs[i].UID == owner.GetUID() {
			return true, refs[i].Controller != nil && *refs[i].Controller
		}
	}
	return false, false
}

func IsOwnerOfGroup(ctrl *metav1.OwnerReference, group string) (bool, string, error) {
	if ctrl == nil {
		return false, "", nil
	}
	gv, err := schema.ParseGroupVersion(ctrl.APIVersion)
	if err != nil {
		return false, "", err
	}
	if gv.Group != group {
		return false, "", nil
	}
	return true, ctrl.Kind, nil
}

func IsOwnerOfGroupKind(ctrl *metav1.OwnerReference, group, kind string) (bool, error) {
	if ctrl == nil {
		return false, nil
	}
	gv, err := schema.ParseGroupVersion(ctrl.APIVersion)
	if err != nil {
		return false, err
	}
	if gv.Group != group {
		return false, nil
	}
	if ctrl.Kind != kind {
		return false, nil
	}
	return true, nil
}

func UpsertToleration(tolerations []core.Toleration, upsert core.Toleration) []core.Toleration {
	for i, toleration := range tolerations {
		if toleration.Key == upsert.Key {
			tolerations[i] = upsert
			return tolerations
		}
	}
	return append(tolerations, upsert)
}

func RemoveToleration(tolerations []core.Toleration, key string) []core.Toleration {
	for i, toleration := range tolerations {
		if toleration.Key == key {
			return append(tolerations[:i], tolerations[i+1:]...)
		}
	}
	return tolerations
}

func UpsertImagePullSecrets(refs []core.LocalObjectReference, upsert ...core.LocalObjectReference) []core.LocalObjectReference {
	for i := range upsert {
		var found bool
		for j := range refs {
			if refs[j].Name == upsert[i].Name {
				found = true
				break
			}
		}
		if !found {
			refs = append(refs, upsert[i])
		}
	}
	return refs
}
