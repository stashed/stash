package controller

import (
	"testing"
	"time"

	rapi "github.com/appscode/restik/api"
	"github.com/appscode/restik/client/clientset"
	rfake "github.com/appscode/restik/client/clientset/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake "k8s.io/client-go/kubernetes/fake"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

var restikName = "appscode-restik"

var fakeRc = &apiv1.ReplicationController{
	TypeMeta: metav1.TypeMeta{
		Kind:       "ReplicationController",
		APIVersion: "v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "appscode-rc",
		Namespace: "default",
		Labels: map[string]string{
			"backup.appscode.com/config": restikName,
		},
	},
	Spec: apiv1.ReplicationControllerSpec{
		Replicas: 1,
		Selector: map[string]string{
			"app": "nginx",
		},
		Template: &apiv1.PodTemplateSpec{
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
					},
				},
			},
		},
	},
}
var fakeRestik = &rapi.Restik{
	TypeMeta: metav1.TypeMeta{
		Kind:       clientset.ResourceKindRestik,
		APIVersion: "backup.appscode.com/v1alpha1",
	},
	ObjectMeta: metav1.ObjectMeta{
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
			Volume: apiv1.Volume{
				Name: "restik-volume",
				VolumeSource: apiv1.VolumeSource{
					AWSElasticBlockStore: &apiv1.AWSElasticBlockStoreVolumeSource{
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
	_, err := fakeController.Clientset.Core().ReplicationControllers("default").Create(fakeRc)
	assert.Nil(t, err)
	b, err := fakeController.ExtClientset.Restiks("default").Create(fakeRestik)
	assert.Nil(t, err)
	err = fakeController.updateObjectAndStartBackup(b)
	assert.Nil(t, err)
}

func TestUpdateObjectAndStopBackup(t *testing.T) {
	fakeController := getFakeController()
	_, err := fakeController.Clientset.Core().ReplicationControllers("default").Create(fakeRc)
	assert.Nil(t, err)
	b, err := fakeController.ExtClientset.Restiks("default").Create(fakeRestik)
	assert.Nil(t, err)
	err = fakeController.updateObjectAndStopBackup(b)
	assert.Nil(t, err)
}

func TestUpdateImage(t *testing.T) {
	fakeController := getFakeController()
	_, err := fakeController.Clientset.Core().ReplicationControllers("default").Create(fakeRc)
	assert.Nil(t, err)
	b, err := fakeController.ExtClientset.Restiks("default").Create(fakeRestik)
	assert.Nil(t, err)
	err = fakeController.updateImage(b, "appscode/restik:fakelatest")
	assert.Nil(t, err)
}

func getFakeController() *Controller {
	fakeController := &Controller{
		Clientset:    fake.NewSimpleClientset(),
		ExtClientset: rfake.NewFakeRestikClient(),
		SyncPeriod:   time.Minute * 2,
		Tag:          "canary",
	}
	return fakeController
}
