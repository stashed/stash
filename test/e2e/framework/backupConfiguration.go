package framework

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/apis/stash/v1beta1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Invocation) BackupConfiguration(repoName string, deploymentName string) v1beta1.BackupConfiguration {
	return v1beta1.BackupConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(f.app),
			Namespace: f.namespace,
		},
		Spec: v1beta1.BackupConfigurationSpec{
			Repository: core.LocalObjectReference{
				Name: repoName,
			},
			Schedule: "*/1 * * * *",
			Target: &v1beta1.Target{
				Ref: v1beta1.TargetRef{
					APIVersion: "apps/v1",
					Kind:       apis.KindDeployment,
					Name:       deploymentName,
				},
				Directories: []string{
					TestSourceDataMountPath,
				},
				VolumeMounts: []core.VolumeMount{
					{
						Name:      TestSourceDataVolumeName,
						MountPath: TestSourceDataMountPath,
					},
				},
			},
			RetentionPolicy: v1alpha1.RetentionPolicy{
				Name:     "keep-last-5",
				KeepLast: 5,
				Prune:    true,
			},
		},
	}
}

func (f *Invocation) CreateBackupConfiguration(backupCfg v1beta1.BackupConfiguration) error {
	_, err := f.StashClient.StashV1beta1().BackupConfigurations(backupCfg.Namespace).Create(&backupCfg)
	return err
}

func (f *Invocation) DeleteBackupConfiguration(backupCfg v1beta1.BackupConfiguration) error {
	return f.StashClient.StashV1beta1().BackupConfigurations(backupCfg.Namespace).Delete(backupCfg.Name, &v1.DeleteOptions{})
}
