package controller

import (
	"errors"

	rapi "github.com/appscode/k8s-addons/api"
	"github.com/appscode/log"
	"github.com/ghodss/yaml"
	"k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/record"
	"k8s.io/kubernetes/pkg/labels"
)

func getKubeObject(kubeClient clientset.Interface, namespace string, ls labels.Selector) ([]byte, string, error) {
	rcs, err := kubeClient.Core().ReplicationControllers(namespace).List(api.ListOptions{LabelSelector: ls})
	if err == nil && len(rcs.Items) != 0 {
		b, err := yaml.Marshal(rcs.Items[0])
		return b, ReplicationController, err
	}

	replicasets, err := kubeClient.Extensions().ReplicaSets(namespace).List(api.ListOptions{LabelSelector: ls})
	if err == nil && len(replicasets.Items) != 0 {
		b, err := yaml.Marshal(replicasets.Items[0])
		return b, ReplicaSet, err
	}

	deployments, err := kubeClient.Extensions().Deployments(namespace).List(api.ListOptions{LabelSelector: ls})
	if err == nil && len(deployments.Items) != 0 {
		b, err := yaml.Marshal(deployments.Items[0])
		return b, Deployment, err
	}

	daemonsets, err := kubeClient.Extensions().DaemonSets(namespace).List(api.ListOptions{LabelSelector: ls})
	if err == nil && len(daemonsets.Items) != 0 {
		b, err := yaml.Marshal(daemonsets.Items[0])
		return b, DaemonSet, err
	}

	statefulsets, err := kubeClient.Apps().StatefulSets(namespace).List(api.ListOptions{LabelSelector: ls})
	if err == nil && len(statefulsets.Items) != 0 {
		b, err := yaml.Marshal(statefulsets.Items[0])
		return b, StatefulSet, err
	}
	return nil, "", errors.New("Workload not found")
}

func getRestikContainer(b *rapi.Backup, containerImage string) api.Container {
	container := api.Container{
		Name:            ContainerName,
		Image:           containerImage,
		ImagePullPolicy: api.PullAlways,
		Env: []api.EnvVar{
			{
				Name:  RestikNamespace,
				Value: b.Namespace,
			},
			{
				Name:  RestikResourceName,
				Value: b.Name,
			},
		},
	}
	container.Args = append(container.Args, "watch")
	container.Args = append(container.Args, "--v=10")
	backupVolumeMount := api.VolumeMount{
		Name:      b.Spec.Destination.Volume.Name,
		MountPath: b.Spec.Destination.Path,
	}
	sourceVolumeMount := api.VolumeMount{
		Name:      b.Spec.Source.VolumeName,
		MountPath: b.Spec.Source.Path,
	}
	container.VolumeMounts = append(container.VolumeMounts, backupVolumeMount)
	container.VolumeMounts = append(container.VolumeMounts, sourceVolumeMount)
	return container
}

func (pl *Controller) addAnnotation(b *rapi.Backup) {
	if b.ObjectMeta.Annotations == nil {
		b.ObjectMeta.Annotations = make(map[string]string)
	}
	b.ObjectMeta.Annotations[ImageAnnotation] = pl.Image
}

func findSelectors(lb map[string]string) labels.Selector {
	set := labels.Set(lb)
	selectors := labels.SelectorFromSet(set)
	return selectors
}

func restartPods(kubeClient clientset.Interface, namespace string, opts api.ListOptions) error {
	pods, err := kubeClient.Core().Pods(namespace).List(opts)
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		deleteOpts := &api.DeleteOptions{}
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
		func(event *api.Event) {
			if _, err := client.Core().Events(event.Namespace).Create(event); err != nil {
				log.Errorln(err)
			}
		},
	)
	// Event Recorder
	return broadcaster.NewRecorder(api.EventSource{Component: component})
}

func removeContainer(c []api.Container, name string) []api.Container {
	for i, v := range c {
		if v.Name == name {
			c = append(c[:i], c[i+1:]...)
			break
		}
	}
	return c
}
func updateImageForRestikContainer(c []api.Container, name, image string) []api.Container {
	for i, v := range c {
		if v.Name == name {
			c[i].Image = image
			break
		}
	}
	return c
}

func removeVolume(volumes []api.Volume, name string) []api.Volume {
	for i, v := range volumes {
		if v.Name == name {
			volumes = append(volumes[:i], volumes[i+1:]...)
			break
		}
	}
	return volumes
}
