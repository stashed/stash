package e2e_test

import (
	"fmt"

	"github.com/appscode/go/types"
	"github.com/appscode/log"
	sapi "github.com/appscode/stash/api"
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

func createTestNamespace(ctrl *controller.Controller, name string) error {
	namespace = name
	ns := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err := ctrl.kubeClient.CoreV1().Namespaces().Create(ns)
	return err
}

func deleteTestNamespace(watcher *controller.Controller, name string) {
	if err := watcher.kubeClient.CoreV1().Namespaces().Delete(name, &metav1.DeleteOptions{}); err != nil {
		fmt.Println(err)
	}
}

func createReplicationController(watcher *controller.Controller, name string, backupName string) error {
	kubeClient := watcher.kubeClient
	rc := &apiv1.ReplicationController{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ReplicationController",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				sapi.ConfigName: backupName,
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

func deleteReplicationController(ctrl *controller.Controller, name string) {
	if err := ctrl.kubeClient.CoreV1().ReplicationControllers(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createSecret(ctrl *controller.Controller, name string) error {
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
	_, err := ctrl.kubeClient.CoreV1().Secrets(namespace).Create(secret)
	return err
}

func deleteSecret(ctrl *controller.Controller, name string) {
	if err := ctrl.kubeClient.CoreV1().Secrets(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createRestic(ctrl *controller.Controller, backupName string, secretName string) error {
	stash := &sapi.Restic{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "stash.appscode.com/v1alpha1",
			Kind:       clientset.ResourceKindRestic,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupName,
			Namespace: namespace,
		},
		Spec: sapi.ResticSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "stash-e2e",
				},
			},
			FileGroups: []sapi.FileGroup{
				{
					Path: "/source_path",
					RetentionPolicy: sapi.RetentionPolicy{
						KeepLastSnapshots: 5,
					},
				},
			},
			Schedule: "* * * * * *",
			Backend: sapi.Backend{
				RepositorySecretName: secretName,
				Local: &sapi.LocalSpec{
					Path: "/repo_path",
					Volume: apiv1.Volume{
						Name: "stash-vol",
						VolumeSource: apiv1.VolumeSource{
							EmptyDir: &apiv1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		},
	}
	_, err := ctrl.stashClient.Restics(namespace).Create(stash)
	return err
}

func deleteRestic(ctrl *controller.Controller, stashName string) error {
	return ctrl.stashClient.Restics(namespace).Delete(stashName, nil)
}

func createReplicaset(ctrl *controller.Controller, name string, stashName string) error {
	replicaset := &extensions.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "stash-e2e",
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
	_, err := ctrl.kubeClient.ExtensionsV1beta1().ReplicaSets(namespace).Create(replicaset)
	return err
}

func deleteReplicaset(watcher *controller.Controller, name string) {
	if err := watcher.kubeClient.ExtensionsV1beta1().ReplicaSets(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createDeployment(ctrl *controller.Controller, name string, stashName string) error {
	deployment := &extensions.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "stash-e2e",
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
	_, err := ctrl.kubeClient.ExtensionsV1beta1().Deployments(namespace).Create(deployment)
	return err
}

func deleteDeployment(ctrl *controller.Controller, name string) {
	if err := ctrl.kubeClient.ExtensionsV1beta1().Deployments(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createDaemonsets(ctrl *controller.Controller, name string, backupName string) error {
	daemonset := &extensions.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "stash-e2e",
			},
		},
		Spec: extensions.DaemonSetSpec{
			Template: *podTemplate,
		},
	}
	_, err := ctrl.kubeClient.ExtensionsV1beta1().DaemonSets(namespace).Create(daemonset)
	return err
}

func deleteDaemonset(ctrl *controller.Controller, name string) {
	if err := ctrl.kubeClient.ExtensionsV1beta1().DaemonSets(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createStatefulSet(ctrl *controller.Controller, name string, stashName string, svc string) error {
	s := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "stash-e2e",
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
	_, err := ctrl.kubeClient.AppsV1beta1().StatefulSets(namespace).Create(s)
	return err
}

func deleteStatefulset(ctrl *controller.Controller, name string) {
	if err := ctrl.kubeClient.AppsV1beta1().StatefulSets(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		log.Errorln(err)
	}
}

func createService(ctrl *controller.Controller, name string) error {
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
	_, err := ctrl.kubeClient.CoreV1().Services(namespace).Create(svc)
	return err
}

func deleteService(ctrl *controller.Controller, name string) {
	err := ctrl.kubeClient.CoreV1().Services(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		log.Errorln(err)
	}
}
