package framework

import (
	"github.com/appscode/go/crypto/rand"
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/client/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func (f *Framework) Restic() sapi.Restic {
	return sapi.Restic{
		TypeMeta: metav1.TypeMeta{
			APIVersion: sapi.SchemeGroupVersion.String(),
			Kind:       clientset.ResourceKindRestic,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix("stash"),
			Namespace: f.namespace,
		},
		Spec: sapi.ResticSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "stash-e2e",
				},
			},
			FileGroups: []sapi.FileGroup{
				{
					Path: "/lib",
					RetentionPolicy: sapi.RetentionPolicy{
						KeepLastSnapshots: 5,
					},
				},
			},
			Schedule: "@every 10m",
			Backend: sapi.Backend{
				RepositorySecretName: "",
				Local: &sapi.LocalSpec{
					Path: "/repo",
					Volume: apiv1.Volume{
						Name: "repo",
						VolumeSource: apiv1.VolumeSource{
							EmptyDir: &apiv1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		},
	}
}

func (f *Framework) CreateRestic(obj sapi.Restic) error {
	_, err := f.stashClient.Restics(obj.Namespace).Create(&obj)
	return err
}

func (f *Framework) DeleteRestic(meta metav1.ObjectMeta) error {
	return f.stashClient.Restics(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{})
}
