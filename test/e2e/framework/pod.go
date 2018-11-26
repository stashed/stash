package framework

import (
	"bytes"
	"fmt"
	"strings"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TestSourceDataVolumeName = "source-data"
	TestSourceDataMountPath  = "/source/data"
	OperatorNamespace        = "kube-system"
	OperatorName             = "stash-operator"
)

func (fi *Invocation) PodTemplate(labels map[string]string) core.PodTemplateSpec {
	return core.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
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

func (f *Framework) GetPod(meta metav1.ObjectMeta) (*core.Pod, error) {
	podList, err := f.KubeClient.CoreV1().Pods(meta.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, pod := range podList.Items {
		if bytes.HasPrefix([]byte(pod.Name), []byte(meta.Name)) {
			return &pod, nil
		}
	}
	return nil, fmt.Errorf("no pod found for workload %v", meta.Name)
}

func (f *Framework) GetOperatorPod() (*core.Pod, error) {
	podList, err := f.KubeClient.CoreV1().Pods(OperatorNamespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, pod := range podList.Items {
		if strings.HasPrefix(pod.Name, OperatorName) {
			return &pod, nil
		}
	}
	return nil, fmt.Errorf("operator pod not found")
}
