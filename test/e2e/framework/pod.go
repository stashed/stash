package framework

import (
	"bytes"

	. "github.com/onsi/gomega"
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

func (f *Framework) EventuallyPod(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(func() *core.Pod {
		podList, err := f.KubeClient.CoreV1().Pods(meta.Namespace).List(metav1.ListOptions{})
		if err != nil {
			return nil
		}
		for _, pod := range podList.Items {
			if bytes.HasPrefix([]byte(pod.Name), []byte(meta.Name)) {
				return &pod
			}
		}
		return nil
	})
}
