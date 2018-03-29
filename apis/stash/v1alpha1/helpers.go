package v1alpha1

import (
	"hash/fnv"
	"strconv"

	core "k8s.io/api/core/v1"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
)

func (l LocalSpec) ToVolumeAndMount(volName string) (core.Volume, core.VolumeMount) {
	vol := core.Volume{
		Name:         volName,
		VolumeSource: *l.VolumeSource.DeepCopy(), // avoid defaulting in MutatingWebhook
	}
	mnt := core.VolumeMount{
		Name:      volName,
		MountPath: l.MountPath,
		SubPath:   l.SubPath,
	}
	return vol, mnt
}

func (r Restic) GetSpecHash() string {
	hash := fnv.New64a()
	hashutil.DeepHashObject(hash, r.Spec)
	return strconv.FormatUint(hash.Sum64(), 10)
}
