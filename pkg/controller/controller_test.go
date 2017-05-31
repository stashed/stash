package controller

import (
	"testing"
	"time"

	rapi "github.com/appscode/restik/api"
	"github.com/appscode/restik/client/clientset"
	"github.com/appscode/restik/client/clientset/fake"
	"github.com/stretchr/testify/assert"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	fakeclientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
)

var restikName = "appscode-restik"

var fakeRc = &api.ReplicationController{
	TypeMeta: unversioned.TypeMeta{
		Kind:       "ReplicationController",
		APIVersion: "v1",
	},
	ObjectMeta: api.ObjectMeta{
		Name:      "appscode-rc",
		Namespace: "default",
		Labels: map[string]string{
			"backup.appscode.com/config": restikName,
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
var fakeRestik = &rapi.Restik{
	TypeMeta: unversioned.TypeMeta{
		Kind:       clientset.ResourceKindRestik,
		APIVersion: "backup.appscode.com/v1beta1",
	},
	ObjectMeta: api.ObjectMeta{
		Name:      restikName,
		Namespace: "default",
	},
	Spec: rapi.RestikSpec{
		Source: rapi.Source{
			VolumeName: "volume-test",
			Path:       "/mypath",
		},
		Destination: rapi.Destination{
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
	fakeController := getFakeController()
	_, err := fakeController.Client.Core().ReplicationControllers("default").Create(fakeRc)
	assert.Nil(t, err)
	b, err := fakeController.ExtClient.Restiks("default").Create(fakeRestik)
	assert.Nil(t, err)
	err = fakeController.updateObjectAndStartBackup(b)
	assert.Nil(t, err)
}

func TestUpdateObjectAndStopBackup(t *testing.T) {
	fakeController := getFakeController()
	_, err := fakeController.Client.Core().ReplicationControllers("default").Create(fakeRc)
	assert.Nil(t, err)
	b, err := fakeController.ExtClient.Restiks("default").Create(fakeRestik)
	assert.Nil(t, err)
	err = fakeController.updateObjectAndStopBackup(b)
	assert.Nil(t, err)
}

func TestUpdateImage(t *testing.T) {
	fakeController := getFakeController()
	_, err := fakeController.Client.Core().ReplicationControllers("default").Create(fakeRc)
	assert.Nil(t, err)
	b, err := fakeController.ExtClient.Restiks("default").Create(fakeRestik)
	assert.Nil(t, err)
	err = fakeController.updateImage(b, "appscode/restik:fakelatest")
	assert.Nil(t, err)
}

func getFakeController() *Controller {
	fakeController := &Controller{
		Client:     fakeclientset.NewSimpleClientset(),
		ExtClient:  fake.NewFakeExtensionClient(),
		SyncPeriod: time.Minute * 2,
		Image:      "appscode/restik:fake",
	}
	return fakeController
}
