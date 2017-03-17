package controller

import (
	"testing"
	"time"

	rapi "github.com/appscode/restik/api"
	"github.com/appscode/restik/client/clientset/fake"
	"github.com/stretchr/testify/assert"
	"k8s.io/kubernetes/pkg/api"
	fakeclientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
)

var backupName = "appscode-backup"

var fakeRc = &api.ReplicationController{
	ObjectMeta: api.ObjectMeta{
		Name:      "appscode-rc",
		Namespace: "default",
		Labels: map[string]string{
			" backup.appscode.com/config": backupName,
		},
	},
	Spec: api.ReplicationControllerSpec{
		Replicas: 1,
		Selector: map[string]string{
			"app": "nginx",
		},
		Template: &api.PodTemplateSpec{
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
		},
	},
}
var fakeBackup = &rapi.Backup{
	ObjectMeta: api.ObjectMeta{
		Name:      backupName,
		Namespace: "default",
	},
	Spec: rapi.BackupSpec{
		Source: rapi.BackupSource{
			VolumeName: "volume-test",
			Path:       "/mypath",
		},
		Destination: rapi.BackupDestination{
			Path:                 "/restik_repo",
			RepositorySecretName: "restik-secret",
			Volume: api.Volume{
				Name: "restik-volume",
				VolumeSource: api.VolumeSource{
					AWSElasticBlockStore: &api.AWSElasticBlockStoreVolumeSource{
						FSType:   "ext4",
						VolumeID: "vol-12345",
					},
				},
			},
		},
		Schedule: "* * * * * *",
		RetentionPolicy: rapi.RetentionPolicy{
			KeepLastSnapshots: 10,
		},
	},
}

func TestUpdateObjectAndStartBackup(t *testing.T) {
	fakeController := &Controller{
		Client:     fakeclientset.NewSimpleClientset(),
		ExtClient:  fake.NewFakeExtensionClient(),
		SyncPeriod: time.Minute * 2,
		Image:      "appscode/restik:fake",
	}
	_, err := fakeController.Client.Core().ReplicationControllers("default").Create(fakeRc)
	assert.Nil(t, err)
	b, err := fakeController.ExtClient.Backups("default").Create(fakeBackup)
	assert.Nil(t, err)
	err = fakeController.updateObjectAndStartBackup(b)
	assert.Nil(t, err)
}
