package test

import (
	rapi "github.com/appscode/restik/api"
	"github.com/appscode/restik/pkg/controller"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

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
			},
		},
	},
}
var namespace = "default"

func createReplicationController(watcher *controller.Controller,name string) error {
	kubeClient := watcher.Client
	rc := &api.ReplicationController{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "v1",
			Kind: "ReplicationController",
		},
		ObjectMeta: api.ObjectMeta{
			Name: name,
			Namespace: "default",
			Labels: map[string]string{
				"backup.appscode.com/config": "backup-rc",
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

func deleteReplicationController(watcher *controller.Controller, name string) error {
	 return watcher.Client.Core().ReplicationControllers(namespace).Delete(name, &api.DeleteOptions{})

}

func createSecret(watcher *controller.Controller) error {
	secret := &api.Secret{
		TypeMeta: unversioned.TypeMeta{
			APIVersion:"v1",
			Kind: "Secret",
		},
		ObjectMeta:api.ObjectMeta{
			Name: "restik_secret",
			Namespace:namespace,
		},
		Data:map[string][]byte {
			"password": []byte("appscode"),
		},
	}
	_, err := watcher.Client.Core().Secrets(namespace).Create(secret)
	return err
}

func deleteSecret(watcher *controller.Controller) error {
	return watcher.Client.Core().Secrets(namespace).Delete( "restik_secret", &api.DeleteOptions{})
}

func createBackup(watcher *controller.Controller, backupName string) error {
	backup := &rapi.Backup{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "appscode.com/v1beta1",
			Kind: "Backup",
		},
		ObjectMeta: api.ObjectMeta{
			Name: backupName,
			Namespace: namespace,
		},
		Spec: rapi.BackupSpec{
			Source: rapi.BackupSource{
				Path: "/source_path",
				VolumeName: "test-volume",
			},
			Schedule: "* * * * * *",
			Destination: rapi.BackupDestination{
				Path: "/repo_path",
				RepositorySecretName: "backup_secret",
				Volume: api.Volume{
					Name: "restik_vol",
					VolumeSource:api.VolumeSource{
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
	return watcher.ExtClient.Backups(namespace).Delete(backupName,&api.DeleteOptions{})
}

func createReplicaset(watcher *controller.Controller) error {
	replicaset := &extensions.ReplicaSet{
		ObjectMeta: api.ObjectMeta{
			Name:      "appscode-rs",
			Namespace: "default",
			Labels: map[string]string{
				"backup.appscode.com/config": "backup-rc",
			},
		},
/*		Spec: extensions.ReplicaSet{
			Replicas: 1,
			Template: podTemplate,
		},*/
	}
	_, err := watcher.Client.Extensions().ReplicaSets(namespace).Create(replicaset)
	return err
}
