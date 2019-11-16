/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"time"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (f *Invocation) GetBackupConfigurationForWorkload(repoName string, targetRef v1beta1.TargetRef) *v1beta1.BackupConfiguration {
	return &v1beta1.BackupConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rand.WithUniqSuffix(f.app),
			Namespace: f.namespace,
		},
		Spec: v1beta1.BackupConfigurationSpec{
			Repository: core.LocalObjectReference{
				Name: repoName,
			},
			// some workloads such as StatefulSet or DaemonSet may take long to complete backup. so, giving a fixed short interval is not always feasible.
			// hence, set the schedule to a large interval so that no backup schedule appear before completing running backup
			// we will use manual triggering for taking backup. this will help us to avoid waiting for fixed interval before each backup
			// and the problem mentioned above
			Schedule: "59 * * * *",
			RetentionPolicy: v1alpha1.RetentionPolicy{
				Name:     "keep-last-5",
				KeepLast: 5,
				Prune:    true,
			},
			BackupConfigurationTemplateSpec: v1beta1.BackupConfigurationTemplateSpec{
				Target: &v1beta1.BackupTarget{
					Ref: targetRef,
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
			},
		},
	}
}

func (f *Invocation) CreateBackupConfiguration(backupCfg v1beta1.BackupConfiguration) error {
	_, err := f.StashClient.StashV1beta1().BackupConfigurations(backupCfg.Namespace).Create(&backupCfg)
	return err
}

func (f *Invocation) DeleteBackupConfiguration(backupCfg v1beta1.BackupConfiguration) error {
	err := f.StashClient.StashV1beta1().BackupConfigurations(backupCfg.Namespace).Delete(backupCfg.Name, &metav1.DeleteOptions{})
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Invocation) PVCBackupTarget(pvcName string) *v1beta1.BackupTarget {
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

func (f *Framework) EventuallyCronJobCreated(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			_, err := f.KubeClient.BatchV1beta1().CronJobs(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
			if err == nil && !kerr.IsNotFound(err) {
				return true
			}
			return false
		},
		time.Minute*2,
		time.Second*5,
	)
}

func (f *Framework) EventuallyBackupConfigurationCreated(meta metav1.ObjectMeta) GomegaAsyncAssertion {
	return Eventually(
		func() bool {
			_, err := f.StashClient.StashV1beta1().BackupConfigurations(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
			if err == nil && !kerr.IsNotFound(err) {
				return true
			}
			return false
		},
		time.Minute*2,
		time.Second*5,
	)
}
