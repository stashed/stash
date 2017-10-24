package framework

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TestSourceDataVolumeName = "source-data"
	TestSourceDataMountPath  = "/source/data"
)

func (fi *Invocation) PodTemplate() core.PodTemplateSpec {
	return core.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": fi.app,
			},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:            "busybox",
					Image:           "busybox",
					ImagePullPolicy: core.PullIfNotPresent,
					Command: []string{
						"sleep",
						"3600",
					},
					VolumeMounts: []core.VolumeMount{
						{
							Name:      TestSourceDataVolumeName,
							MountPath: TestSourceDataMountPath,
						},
					},
				},
			},
			Volumes: []core.Volume{
				{
					Name: TestSourceDataVolumeName,
					VolumeSource: core.VolumeSource{
						GitRepo: &core.GitRepoVolumeSource{
							Repository: "https://github.com/appscode/stash-data.git",
						},
					},
				},
			},
		},
	}
}
