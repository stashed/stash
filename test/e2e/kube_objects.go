package test

import (
	"fmt"

	"github.com/appscode/go/types"
	"github.com/appscode/log"
	api "github.com/appscode/stash/api"
	"github.com/appscode/stash/client/clientset"
	"github.com/appscode/stash/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

var namespace string
var podTemplate = &apiv1.PodTemplateSpec{
	ObjectMeta: metav1.ObjectMeta{
		Name: "nginx",
		Labels: map[string]string{
			"app": "nginx",
		},
	},
	Spec: apiv1.PodSpec{
		Containers: []apiv1.Container{
			{
				Name:  "nginx",
				Image: "nginx",
				VolumeMounts: []apiv1.VolumeMount{
					{
						Name:      "test-volume",
						MountPath: "/source_path",
					},
				},
			},
		},
		Volumes: []apiv1.Volume{
			{
				Name: "test-volume",
				VolumeSource: apiv1.VolumeSource{
					EmptyDir: &apiv1.EmptyDirVolumeSource{},
				},
			},
		},
	},
}

func createTestNamespace(watcher *controller.Controller, name string) error {
	namespace = name
	ns := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err := watcher.KubeClient.CoreV1().Namespaces().Create(ns)
	return err
}

func deleteTestNamespace(watcher *controller.Controller, name string) {
	if err := watcher.KubeClient.CoreV1().Namespaces().Delete(name, &metav1.DeleteOptions{}); err != nil {
		fmt.Println(err)
	}
}

func createReplicationController(watcher *controller.Controller, name string, backupName string) error {
	kubeClient := watcher.KubeClient
	rc := &apiv1.ReplicationController{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ReplicationController",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				controller.ConfigName: backupName,
			},
		},
		Spec: apiv1.ReplicationControllerSpec{
			Replicas: types.Int32P(1),
			Template: podTemplate,
		},
	}
	_, err := kubeClient.CoreV1().ReplicationControllers(namespace).Create(rc)
	return err
}

func deleteReplicationController(watcher *controller.Controller, name string) {
	if err := watcher.KubeClient.CoreV1().ReplicationControllers(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createSecret(watcher *controller.Controller, name string) error {
	secret := &apiv1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"password": []byte("appscode"),
		},
	}
	_, err := watcher.KubeClient.CoreV1().Secrets(namespace).Create(secret)
	return err
}

func deleteSecret(watcher *controller.Controller, name string) {
	if err := watcher.KubeClient.CoreV1().Secrets(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createStash(watcher *controller.Controller, backupName string, secretName string) error {
	stash := &api.Restic{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "stash.appscode.com/v1alpha1",
			Kind:       clientset.ResourceKindRestic,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupName,
			Namespace: namespace,
		},
		Spec: api.ResticSpec{
			Source: api.Source{
				Path:       "/source_path",
				VolumeName: "test-volume",
			},
			Schedule: "* * * * * *",
			Destination: api.Destination{
				Path:                 "/repo_path",
				RepositorySecretName: secretName,
				Volume: apiv1.Volume{
					Name: "stash-vol",
					VolumeSource: apiv1.VolumeSource{
						EmptyDir: &apiv1.EmptyDirVolumeSource{},
					},
				},
			},
			RetentionPolicy: api.RetentionPolicy{
				KeepLastSnapshots: 5,
			},
		},
	}
	_, err := watcher.StashClient.Restics(namespace).Create(stash)
	return err
}

func deleteStash(watcher *controller.Controller, stashName string) error {
	return watcher.StashClient.Restics(namespace).Delete(stashName, nil)
}

func createReplicaset(watcher *controller.Controller, name string, stashName string) error {
	replicaset := &extensions.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				controller.ConfigName: stashName,
			},
		},
		Spec: extensions.ReplicaSetSpec{
			Replicas: types.Int32P(1),
			Template: *podTemplate,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
		},
	}
	_, err := watcher.KubeClient.ExtensionsV1beta1().ReplicaSets(namespace).Create(replicaset)
	return err
}

func deleteReplicaset(watcher *controller.Controller, name string) {
	if err := watcher.KubeClient.ExtensionsV1beta1().ReplicaSets(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createDeployment(watcher *controller.Controller, name string, stashName string) error {
	deployment := &extensions.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				controller.ConfigName: stashName,
			},
		},
		Spec: extensions.DeploymentSpec{
			Replicas: types.Int32P(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			Template: *podTemplate,
		},
	}
	_, err := watcher.KubeClient.ExtensionsV1beta1().Deployments(namespace).Create(deployment)
	return err
}

func deleteDeployment(watcher *controller.Controller, name string) {
	if err := watcher.KubeClient.ExtensionsV1beta1().Deployments(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createDaemonsets(watcher *controller.Controller, name string, backupName string) error {
	daemonset := &extensions.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				controller.ConfigName: backupName,
			},
		},
		Spec: extensions.DaemonSetSpec{
			Template: *podTemplate,
		},
	}
	_, err := watcher.KubeClient.ExtensionsV1beta1().DaemonSets(namespace).Create(daemonset)
	return err
}

func deleteDaemonset(watcher *controller.Controller, name string) {
	if err := watcher.KubeClient.ExtensionsV1beta1().DaemonSets(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createStatefulSet(watcher *controller.Controller, name string, stashName string, svc string) error {
	s := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				controller.ConfigName: stashName,
			},
		},
		Spec: apps.StatefulSetSpec{
			Replicas:    types.Int32P(1),
			Template:    *podTemplate,
			ServiceName: svc,
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
	_, err := watcher.KubeClient.AppsV1beta1().StatefulSets(namespace).Create(s)
	return err
}

func deleteStatefulset(watcher *controller.Controller, name string) {
	if err := watcher.KubeClient.AppsV1beta1().StatefulSets(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createService(watcher *controller.Controller, name string) error {
	svc := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Spec: apiv1.ServiceSpec{
			Selector: map[string]string{
				"app": "nginx",
			},
			Ports: []apiv1.ServicePort{
				{
					Port: 80,
					Name: "web",
				},
			},
		},
	}
	_, err := watcher.KubeClient.CoreV1().Services(namespace).Create(svc)
	return err
}

func deleteService(watcher *controller.Controller, name string) {
	err := watcher.KubeClient.CoreV1().Services(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		log.Errorln(err)
	}
}
