package framework

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	"github.com/appscode/stash/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func (f *Framework) StatefulSet(namespace string) apps.StatefulSet {
	s := apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: namespace,
			Labels: map[string]string{
				"app": "stash-e2e",
			},
		},
		Spec: apps.StatefulSetSpec{
			Replicas:    types.Int32P(1),
			Template:    &f.PodTemplate(),
			ServiceName: TEST_HEADLESS_SERVICE,
		},
	}
	container := apiv1.Container{
		Name:            controller.ContainerName,
		Image:           image,
		ImagePullPolicy: apiv1.PullIfNotPresent,
		Env: []apiv1.EnvVar{
			{
				Name:  controller.StashNamespace,
				Value: namespace,
			},
			{
				Name:  controller.StashResourceName,
				Value: stashName,
			},
		},
	}
	container.Args = append(container.Args, "watch")
	container.Args = append(container.Args, "--v=10")
	backupVolumeMount := apiv1.VolumeMount{
		Name:      "test-volume",
		MountPath: "/source_path",
	}
	sourceVolumeMount := apiv1.VolumeMount{
		Name:      "stash-vol",
		MountPath: "/repo_path",
	}
	container.VolumeMounts = append(container.VolumeMounts, backupVolumeMount)
	container.VolumeMounts = append(container.VolumeMounts, sourceVolumeMount)
	s.Spec.Template.Spec.Containers = append(s.Spec.Template.Spec.Containers, container)
	s.Spec.Template.Spec.Volumes = append(s.Spec.Template.Spec.Volumes, apiv1.Volume{
		Name: "stash-vol",
		VolumeSource: apiv1.VolumeSource{
			EmptyDir: &apiv1.EmptyDirVolumeSource{},
		},
	})
	return s
}

func (f *Framework) CreateStatefulSet(obj apps.StatefulSet) error {
	_, err := f.KubeClient.AppsV1beta1().StatefulSets(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteStatefulset(meta metav1.ObjectMeta) error {
	return f.KubeClient.AppsV1beta1().StatefulSets(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{})
}
