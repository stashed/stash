package controller

import (
	"errors"

	"github.com/appscode/log"
	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/record"
)

func getKubeObject(kubeClient clientset.Interface, namespace string, ls labels.Selector) ([]byte, string, error) {
	rcs, err := kubeClient.Core().ReplicationControllers(namespace).List(apiv1.ListOptions{LabelSelector: ls})
	if err == nil && len(rcs.Items) != 0 {
		b, err := yaml.Marshal(rcs.Items[0])
		return b, ReplicationController, err
	}

	replicasets, err := kubeClient.Extensions().ReplicaSets(namespace).List(apiv1.ListOptions{LabelSelector: ls})
	if err == nil && len(replicasets.Items) != 0 {
		b, err := yaml.Marshal(replicasets.Items[0])
		return b, ReplicaSet, err
	}

	deployments, err := kubeClient.Extensions().Deployments(namespace).List(apiv1.ListOptions{LabelSelector: ls})
	if err == nil && len(deployments.Items) != 0 {
		b, err := yaml.Marshal(deployments.Items[0])
		return b, Deployment, err
	}

	daemonsets, err := kubeClient.Extensions().DaemonSets(namespace).List(apiv1.ListOptions{LabelSelector: ls})
	if err == nil && len(daemonsets.Items) != 0 {
		b, err := yaml.Marshal(daemonsets.Items[0])
		return b, DaemonSet, err
	}

	statefulsets, err := kubeClient.Apps().StatefulSets(namespace).List(apiv1.ListOptions{LabelSelector: ls})
	if err == nil && len(statefulsets.Items) != 0 {
		b, err := yaml.Marshal(statefulsets.Items[0])
		return b, StatefulSet, err
	}
	return nil, "", errors.New("Workload not found")
}

func getRestikContainer(r *rapiv1.Restik, containerImage string) apiv1.Container {
	container := apiv1.Container{
		Name:            ContainerName,
		Image:           containerImage,
		ImagePullPolicy: apiv1.PullAlways,
		Env: []apiv1.EnvVar{
			{
				Name:  RestikNamespace,
				Value: r.Namespace,
			},
			{
				Name:  RestikResourceName,
				Value: r.Name,
			},
		},
	}
	container.Args = append(container.Args, "watch")
	container.Args = append(container.Args, "--v=10")
	backupVolumeMount := apiv1.VolumeMount{
		Name:      r.Spec.Destination.Volume.Name,
		MountPath: r.Spec.Destination.Path,
	}
	sourceVolumeMount := apiv1.VolumeMount{
		Name:      r.Spec.Source.VolumeName,
		MountPath: r.Spec.Source.Path,
	}
	container.VolumeMounts = append(container.VolumeMounts, backupVolumeMount)
	container.VolumeMounts = append(container.VolumeMounts, sourceVolumeMount)
	return container
}

func (c *Controller) addAnnotation(r *rapiv1.Restik) {
	if r.ObjectMeta.Annotations == nil {
		r.ObjectMeta.Annotations = make(map[string]string)
	}
	r.ObjectMeta.Annotations[ImageAnnotation] = c.Image
}

func findSelectors(lb map[string]string) labels.Selector {
	set := labels.Set(lb)
	selectors := labels.SelectorFromSet(set)
	return selectors
}

func restartPods(kubeClient clientset.Interface, namespace string, opts apiv1.ListOptions) error {
	pods, err := kubeClient.Core().Pods(namespace).List(opts)
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		deleteOpts := &apiv1.DeleteOptions{}
		err = kubeClient.Core().Pods(namespace).Delete(pod.Name, deleteOpts)
		if err != nil {
			return err
		}
	}
	return nil
}
func getPasswordFromSecret(client clientset.Interface, secretName, namespace string) (string, error) {
	secret, err := client.Core().Secrets(namespace).Get(secretName)
	if err != nil {
		return "", err
	}
	password, ok := secret.Data[Password]
	if !ok {
		return "", errors.New("Restic Password not found")
	}
	return string(password), nil
}

func NewEventRecorder(client clientset.Interface, component string) record.EventRecorder {
	// Event Broadcaster
	broadcaster := record.NewBroadcaster()
	broadcaster.StartEventWatcher(
		func(event *apiv1.Event) {
			if _, err := client.Core().Events(event.Namespace).Create(event); err != nil {
				log.Errorln(err)
			}
		},
	)
	// Event Recorder
	return broadcaster.NewRecorder(apiv1.EventSource{Component: component})
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
func updateImageForRestikContainer(c []apiv1.Container, name, image string) []apiv1.Container {
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
