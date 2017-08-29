package v1

import (
	"errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

// ref: https://github.com/coreos/prometheus-operator/blob/c79166fcff3dae7bb8bc1e6bddc81837c2d97c04/pkg/k8sutil/k8sutil.go#L64
// PodRunningAndReady returns whether a pod is running and each container has
// passed it's ready state.
func PodRunningAndReady(pod apiv1.Pod) (bool, error) {
	switch pod.Status.Phase {
	case apiv1.PodFailed, apiv1.PodSucceeded:
		return false, errors.New("pod completed")
	case apiv1.PodRunning:
		for _, cond := range pod.Status.Conditions {
			if cond.Type != apiv1.PodReady {
				continue
			}
			return cond.Status == apiv1.ConditionTrue, nil
		}
		return false, errors.New("pod ready condition not found")
	}
	return false, nil
}

func RestartPods(kubeClient clientset.Interface, namespace string, selector *metav1.LabelSelector) error {
	r, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return err
	}
	return kubeClient.CoreV1().Pods(namespace).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: r.String(),
	})
}

func GetString(m map[string]string, key string) string {
	if m == nil {
		return ""
	}
	return m[key]
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
