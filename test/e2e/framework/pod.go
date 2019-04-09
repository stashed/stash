package framework

import (
	"bytes"
	"fmt"
	"strings"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

const (
	TestSourceDataVolumeName = "source-data"
	TestSourceDataMountPath  = "/source/data"
	TestSafeDataMountPath    = "/safe/data"
	OperatorNamespace        = "kube-system"
	OperatorName             = "stash-operator"
)

func (fi *Invocation) PodTemplate(labels map[string]string, pvcName string) core.PodTemplateSpec {
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
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}
}

func (f *Framework) GetPod(meta metav1.ObjectMeta) (*core.Pod, error) {
	labelSelector := fields.SelectorFromSet(meta.Labels)
	podList, err := f.KubeClient.CoreV1().Pods(meta.Namespace).List(metav1.ListOptions{LabelSelector: labelSelector.String()})
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

func (f *Framework) GetAllPod(meta metav1.ObjectMeta) ([]core.Pod, error) {
	pods := make([]core.Pod,0)
	labelSelector := fields.SelectorFromSet(meta.Labels)
	podList, err := f.KubeClient.CoreV1().Pods(meta.Namespace).List(metav1.ListOptions{LabelSelector: labelSelector.String()})
	if err != nil {
		return nil, err
	}
	for _, pod := range podList.Items {
		if bytes.HasPrefix([]byte(pod.Name), []byte(meta.Name)) {
			pods = append(pods,pod)
		}
	}
	if len(pods) > 0{
		return pods, nil
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
