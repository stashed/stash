package framework

import (
	"fmt"
	"time"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
)

func (f *Invocation) BackupConfiguration(repoName string, targetref v1beta1.TargetRef) v1beta1.BackupConfiguration {
	return v1beta1.BackupConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(f.app),
			Namespace: f.namespace,
		},
		Spec: v1beta1.BackupConfigurationSpec{
			Repository: core.LocalObjectReference{
				Name: repoName,
			},
			Schedule: "*/3 * * * *",
			Target: &v1beta1.BackupTarget{
				Ref: targetref,
				Paths: []string{
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

func (f *Invocation) DeleteBackupConfiguration(meta metav1.ObjectMeta) error {
	if meta.Name == "" {
		return nil
	}
	err := f.StashClient.StashV1beta1().BackupConfigurations(meta.Namespace).Delete(meta.Name, &metav1.DeleteOptions{})
	if kerr.IsNotFound(err) {
		return nil
	}
	return nil
}

func (f *Invocation) PvcBackupTarget(pvcName string) *v1beta1.BackupTarget {
	return &v1beta1.BackupTarget{
		Ref: v1beta1.TargetRef{
			APIVersion: "v1",
			Kind:       apis.KindPersistentVolumeClaim,
			Name:       pvcName,
		},
		VolumeMounts: []core.VolumeMount{
			{
				Name:      TestSourceDataVolumeName,
				MountPath: TestSourceDataMountPath,
			},
		},
		Paths: []string{
			TestSourceDataMountPath,
		},
	}
}

func (f *Framework) EventuallyBackupConfigurationCreated(namespace string) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			objList, err := f.StashClient.StashV1beta1().BackupConfigurations(namespace).List(metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			if len(objList.Items) > 0 {
				return true
			}
			return false
		},
		time.Minute*2,
		time.Second*5,
	)
}

func (f *Framework) GetBackupConfiguration(namespace string) (backupConfig *v1beta1.BackupConfiguration, err error) {
	backupCfglist, err := f.StashClient.StashV1beta1().BackupConfigurations(namespace).List(metav1.ListOptions{})
	if err != nil {
		return backupConfig, err
	}
	if len(backupCfglist.Items) > 0 {
		return &backupCfglist.Items[0], nil
	}
	return backupConfig, fmt.Errorf("no BackupSession found")
}
