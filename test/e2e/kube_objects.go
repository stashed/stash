package test

import (
	rapi "github.com/appscode/restik/api"
	"github.com/appscode/restik/pkg/controller"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"log"
)

var namespace = "sauman"
var podTemplate = &api.PodTemplateSpec{
	ObjectMeta: api.ObjectMeta{
		Name: "nginx",
		Labels: map[string]string{
			"app": "nginx",
		},
	},
	Spec: api.PodSpec{
		Containers: []api.Container{
			{
				Name:  "nginx",
				Image: "nginx",
				VolumeMounts: []api.VolumeMount{
					{
						Name:      "test-volume",
						MountPath: "/source_path",
					},
				},
			},
		},
		Volumes: []api.Volume{
			{
				Name: "test-volume",
				VolumeSource: api.VolumeSource{
					EmptyDir: &api.EmptyDirVolumeSource{},
				},
			},
		},
	},
}

func createReplicationController(watcher *controller.Controller, name string, backupName string) error {
	kubeClient := watcher.Client
	rc := &api.ReplicationController{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "v1",
			Kind:       "ReplicationController",
		},
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"backup.appscode.com/config": backupName,
			},
		},
		Spec: api.ReplicationControllerSpec{
			Replicas: 1,
			Template: podTemplate,
		},
	}
	_, err := kubeClient.Core().ReplicationControllers(namespace).Create(rc)
	return err
}

func deleteReplicationController(watcher *controller.Controller, name string) {
	if err := watcher.Client.Core().ReplicationControllers(namespace).Delete(name, &api.DeleteOptions{}); err != nil {
		log.Println(err)
	}
}

func createSecret(watcher *controller.Controller, name string) error {
	secret := &api.Secret{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"password": []byte("appscode"),
		},
	}
	_, err := watcher.Client.Core().Secrets(namespace).Create(secret)
	return err
}

func deleteSecret(watcher *controller.Controller, name string) {
	if err := watcher.Client.Core().Secrets(namespace).Delete(name, &api.DeleteOptions{}); err != nil {
		log.Println(err)
	}
}

func createBackup(watcher *controller.Controller, backupName string, secretName string) error {
	backup := &rapi.Backup{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "appscode.com/v1beta1",
			Kind:       "Backup",
		},
		ObjectMeta: api.ObjectMeta{
			Name:      backupName,
			Namespace: namespace,
		},
		Spec: rapi.BackupSpec{
			Source: rapi.BackupSource{
				Path:       "/source_path",
				VolumeName: "test-volume",
			},
			Schedule: "* * * * * *",
			Destination: rapi.BackupDestination{
				Path:                 "/repo_path",
				RepositorySecretName: secretName,
				Volume: api.Volume{
					Name: "restik-vol",
					VolumeSource: api.VolumeSource{
						EmptyDir: &api.EmptyDirVolumeSource{},
					},
				},
			},
			RetentionPolicy: rapi.RetentionPolicy{
				KeepLastSnapshots: 5,
			},
		},
	}
	_, err := watcher.ExtClient.Backups(namespace).Create(backup)
	return err
}

func deleteBackup(watcher *controller.Controller, backupName string) error {
	return watcher.ExtClient.Backups(namespace).Delete(backupName, nil)
}

func createReplicaset(watcher *controller.Controller, name string, backupName string) error {
	replicaset := &extensions.ReplicaSet{
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"backup.appscode.com/config": backupName,
			},
		},
		Spec: extensions.ReplicaSetSpec{
			Replicas: 1,
			Template: *podTemplate,
			Selector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
		},
	}
	_, err := watcher.Client.Extensions().ReplicaSets(namespace).Create(replicaset)
	return err
}

func deleteReplicaset(watcher *controller.Controller, name string)  {
	if err:= watcher.Client.Extensions().ReplicaSets(namespace).Delete(name, &api.DeleteOptions{}); err != nil {
		log.Println(err)
	}
}

func deleteEvent(watcher *controller.Controller, name string) {
	if err := watcher.Client.Core().Events(namespace).Delete(name, &api.DeleteOptions{}); err != nil {
		log.Println(err)
	}
}

func createDeployment(watcher *controller.Controller, name string, backupName string) error {
	deployment := &extensions.Deployment{
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"backup.appscode.com/config": backupName,
			},
		},
		Spec: extensions.DeploymentSpec{
			Replicas: 1,
			Selector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			Template: *podTemplate,
		},
	}
	_, err := watcher.Client.Extensions().Deployments(namespace).Create(deployment)
	return err
}

func deleteDeployment(watcher *controller.Controller, name string) {
	if err:= watcher.Client.Extensions().Deployments(namespace).Delete(name, &api.DeleteOptions{}); err != nil {
		log.Println(err)
	}
}

func createDaemonsets(watcher *controller.Controller, name string, backupName string) error {
	daemonset := &extensions.DaemonSet{
		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"backup.appscode.com/config": backupName,
			},
		},
		Spec: extensions.DaemonSetSpec{
			Template: *podTemplate,
		},
	}
	_, err := watcher.Client.Extensions().DaemonSets(namespace).Create(daemonset)
	return err
}

func deleteDaemonset(watcher *controller.Controller, name string) {
	if err := watcher.Client.Extensions().DaemonSets(namespace).Delete(name, &api.DeleteOptions{}); err != nil {
		log.Println(err)
	}
}
