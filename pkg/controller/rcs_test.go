package controller

import (
	"testing"
	"time"

	"github.com/appscode/go/types"
	"github.com/appscode/stash/api"
	"github.com/appscode/stash/client/clientset"
	rfake "github.com/appscode/stash/client/clientset/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake "k8s.io/client-go/kubernetes/fake"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

var stashName = "appscode-stash"

var fakeRc = &apiv1.ReplicationController{
	TypeMeta: metav1.TypeMeta{
		Kind:       "ReplicationController",
		APIVersion: "v1",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "appscode-rc",
		Namespace: "default",
		Labels: map[string]string{
			"restic.appscode.com/config": stashName,
		},
	},
	Spec: apiv1.ReplicationControllerSpec{
		Replicas: types.Int32P(1),
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

var fakeStash = &api.Restic{
	TypeMeta: metav1.TypeMeta{
		Kind:       clientset.ResourceKindRestic,
		APIVersion: api.GroupName,
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      stashName,
		Namespace: "default",
	},
	Spec: api.ResticSpec{
		Source: api.Source{
			VolumeName: "volume-test",
			Path:       "/mypath",
		},
		Backend: api.Backend{
			Path:                 "/stash_repo",
			RepositorySecretName: "stash-secret",
			Volume: apiv1.Volume{
				Name: "stash-volume",
				VolumeSource: apiv1.VolumeSource{
					AWSElasticBlockStore: &apiv1.AWSElasticBlockStoreVolumeSource{
						FSType:   "ext4",
						VolumeID: "vol-12345",
					},
				},
			},
		},
		Schedule: "* * * * * *",
		RetentionPolicy: api.RetentionPolicy{
			KeepLastSnapshots: 10,
		},
	},
}

func TestEnsureReplicationControllerSidecar(t *testing.T) {
	ctrl := getTestController()
	resource, err := ctrl.kubeClient.CoreV1().ReplicationControllers("default").Create(fakeRc)
	assert.Nil(t, err)
	restic, err := ctrl.stashClient.Restics("default").Create(fakeStash)
	assert.Nil(t, err)
	ctrl.EnsureReplicationControllerSidecar(resource, restic)
	assert.Nil(t, err)
}

func TestEnsureReplicationControllerSidecarDeleted(t *testing.T) {
	ctrl := getTestController()
	resource, err := ctrl.kubeClient.CoreV1().ReplicationControllers("default").Create(fakeRc)
	assert.Nil(t, err)
	restic, err := ctrl.stashClient.Restics("default").Create(fakeStash)
	assert.Nil(t, err)
	err = ctrl.EnsureReplicationControllerSidecarDeleted(resource, restic)
	assert.Nil(t, err)
}

func getTestController() *Controller {
	fakeController := &Controller{
		kubeClient:      fake.NewSimpleClientset(),
		stashClient:     rfake.NewFakeStashClient(),
		syncPeriod:      time.Minute * 2,
		SidecarImageTag: "canary",
	}
	return fakeController
}
