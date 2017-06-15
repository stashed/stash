package controller

import (
	"testing"
	"time"

	"github.com/appscode/restik/client/clientset"
	"github.com/appscode/restik/client/clientset/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	fakeclientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
)

var restikName = "appscode-restik"

var fakeRc = &apiv1.ReplicationController{
	TypeMeta: metav1.TypeMeta{
		Kind:       "ReplicationController",
		APIVersion: "v1",
	},
	ObjectMeta: apiv1.ObjectMeta{
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
			ObjectMeta: apiv1.ObjectMeta{
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
var fakeRestik = &rapiv1.Restik{
	TypeMeta: metav1.TypeMeta{
		Kind:       clientset.ResourceKindRestik,
		APIVersion: "backup.appscode.com/v1alpha1",
	},
	ObjectMeta: apiv1.ObjectMeta{
		Name:      restikName,
		Namespace: "default",
	},
	Spec: rapiv1.RestikSpec{
		Source: rapiv1.Source{
			VolumeName: "volume-test",
			Path:       "/mypath",
		},
		Destination: rapiv1.Destination{
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
		RetentionPolicy: rapiv1.RetentionPolicy{
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
		Clientset:    fakeclientset.NewSimpleClientset(),
		ExtClientset: fake.NewFakeRestikClient(),
		SyncPeriod:   time.Minute * 2,
		Image:        "appscode/restik:fake",
	}
	return fakeController
}
