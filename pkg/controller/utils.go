package controller

import (
	"errors"

	rapi "github.com/appscode/stash/api"
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/docker"
	"github.com/ghodss/yaml"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (c *Controller) FindRestic(obj metav1.ObjectMeta) (*sapi.Restic, error) {
	restics, err := c.StashClient.Restics(obj.Namespace).List(metav1.ListOptions{})
	if kerr.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	for _, restic := range restics.Items {
		selector, err := metav1.LabelSelectorAsSelector(&restic.Spec.Selector)
		//return nil, fmt.Errorf("invalid selector: %v", err)
		if err == nil {
			if selector.Matches(labels.Set(obj.Labels)) {
				return &restic, nil
			}
		}
	}
	return nil, nil
}

func (c *Controller) restartPods(namespace string, selector *metav1.LabelSelector) error {
	return c.KubeClient.CoreV1().Pods(namespace).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
}

func getString(m map[string]string, key string) string {
	if m == nil {
		return ""
	}
	return m[key]
}

func getKubeObject(kubeClient clientset.Interface, namespace string, ls labels.Selector) ([]byte, string, error) {
	rcs, err := kubeClient.CoreV1().ReplicationControllers(namespace).List(metav1.ListOptions{LabelSelector: ls.String()})
	if err == nil && len(rcs.Items) != 0 {
		b, err := yaml.Marshal(rcs.Items[0])
		return b, ReplicationController, err
	}

	replicasets, err := kubeClient.ExtensionsV1beta1().ReplicaSets(namespace).List(metav1.ListOptions{LabelSelector: ls.String()})
	if err == nil && len(replicasets.Items) != 0 {
		b, err := yaml.Marshal(replicasets.Items[0])
		return b, ReplicaSet, err
	}

	deployments, err := kubeClient.ExtensionsV1beta1().Deployments(namespace).List(metav1.ListOptions{LabelSelector: ls.String()})
	if err == nil && len(deployments.Items) != 0 {
		b, err := yaml.Marshal(deployments.Items[0])
		return b, Deployment, err
	}

	daemonsets, err := kubeClient.ExtensionsV1beta1().DaemonSets(namespace).List(metav1.ListOptions{LabelSelector: ls.String()})
	if err == nil && len(daemonsets.Items) != 0 {
		b, err := yaml.Marshal(daemonsets.Items[0])
		return b, DaemonSet, err
	}

	statefulsets, err := kubeClient.AppsV1beta1().StatefulSets(namespace).List(metav1.ListOptions{LabelSelector: ls.String()})
	if err == nil && len(statefulsets.Items) != 0 {
		b, err := yaml.Marshal(statefulsets.Items[0])
		return b, StatefulSet, err
	}
	return nil, "", errors.New("Workload not found")
}

func (c *Controller) GetSidecarContainer(r *rapi.Restic) apiv1.Container {
	sidecar := apiv1.Container{
		Name:            docker.StashContainer,
		Image:           docker.ImageOperator + ":" + c.SidecarImageTag,
		ImagePullPolicy: apiv1.PullIfNotPresent,
		Args: []string{
			"crond",
			"--v=3",
			"--namespace=" + r.Namespace,
			"--name=" + r.Name,
		},
		VolumeMounts: []apiv1.VolumeMount{
			{
				Name:      r.Spec.Source.VolumeName,
				MountPath: r.Spec.Source.Path,
			},
		},
	}
	backupVolumeMount := apiv1.VolumeMount{
		Name:      r.Spec.Destination.Volume.Name,
		MountPath: r.Spec.Destination.Path,
	}
	sidecar.VolumeMounts = append(sidecar.VolumeMounts, backupVolumeMount)
	return sidecar
}

func (c *Controller) addAnnotation(r *rapi.Restic) {
	if r.ObjectMeta.Annotations == nil {
		r.ObjectMeta.Annotations = make(map[string]string)
	}
	r.ObjectMeta.Annotations[sapi.VersionTag] = c.SidecarImageTag
}

func findSelectors(lb map[string]string) labels.Selector {
	set := labels.Set(lb)
	selectors := labels.SelectorFromSet(set)
	return selectors
}

func removeContainer(c []apiv1.Container, name string) []apiv1.Container {
	for i, v := range c {
		if v.Name == name {
			c = append(c[:i], c[i+1:]...)
			break
		}
	}
	return c
}
func updateImageForStashContainer(c []apiv1.Container, name, image string) []apiv1.Container {
	for i, v := range c {
		if v.Name == name {
			c[i].Image = image
			break
		}
	}
	return c
}

func removeVolume(volumes []apiv1.Volume, name string) []apiv1.Volume {
	for i, v := range volumes {
		if v.Name == name {
			volumes = append(volumes[:i], volumes[i+1:]...)
			break
		}
	}
	return volumes
}
