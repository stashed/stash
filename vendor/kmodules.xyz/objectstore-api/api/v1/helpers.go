package v1

import (
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
)

func (s Backend) Container() (string, error) {
	if s.S3 != nil {
		return s.S3.Bucket, nil
	} else if s.GCS != nil {
		return s.GCS.Bucket, nil
	} else if s.Azure != nil {
		return s.Azure.Container, nil
	} else if s.Local != nil {
		return s.Local.MountPath, nil
	} else if s.Swift != nil {
		return s.Swift.Container, nil
	}
	return "", errors.New("no storage provider is configured")
}

func (s Backend) Location() (string, error) {
	if s.S3 != nil {
		return "s3:" + s.S3.Bucket, nil
	} else if s.GCS != nil {
		return "gs:" + s.GCS.Bucket, nil
	} else if s.Azure != nil {
		return "azure:" + s.Azure.Container, nil
	} else if s.Local != nil {
		return "local:" + s.Local.MountPath, nil
	} else if s.Swift != nil {
		return "swift:" + s.Swift.Container, nil
	}
	return "", errors.New("no storage provider is configured")
}

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
