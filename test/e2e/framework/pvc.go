/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"context"
	"fmt"
	"strings"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	meta_util "kmodules.xyz/client-go/meta"
)

func (fi *Invocation) PersistentVolumeClaim(name string) *core.PersistentVolumeClaim {
	return &core.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: fi.namespace,
		},
		Spec: core.PersistentVolumeClaimSpec{
			AccessModes: []core.PersistentVolumeAccessMode{
				core.ReadWriteOnce,
			},
			StorageClassName: &fi.StorageClass,
			Resources: core.ResourceRequirements{
				Requests: core.ResourceList{
					core.ResourceStorage: resource.MustParse("10Mi"),
				},
			},
		},
	}
}

func (f *Framework) CreatePersistentVolumeClaim(pvc *core.PersistentVolumeClaim) (*core.PersistentVolumeClaim, error) {
	return f.KubeClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
}

func (fi *Invocation) CreateNewPVC(name string, transformFuncs ...func(p *core.PersistentVolumeClaim)) (*core.PersistentVolumeClaim, error) {
	// Generate PVC definition
	pvc := fi.PersistentVolumeClaim(name)

	for _, fn := range transformFuncs {
		fn(pvc)
	}

	By(fmt.Sprintf("Creating PVC: %s/%s", pvc.Namespace, pvc.Name))
	createdPVC, err := fi.CreatePersistentVolumeClaim(pvc)
	if err != nil {
		return nil, err
	}
	fi.AppendToCleanupList(createdPVC)

	return createdPVC, nil
}

func (fi *Invocation) SetupPVCBackup(pvc *core.PersistentVolumeClaim, repo *v1alpha1.Repository, transformFuncs ...func(bc *v1beta1.BackupConfiguration)) (*v1beta1.BackupConfiguration, error) {
	// Generate desired BackupConfiguration definition
	backupConfig := fi.GetBackupConfiguration(repo.Name, func(bc *v1beta1.BackupConfiguration) {
		bc.Spec.Target = &v1beta1.BackupTarget{
			Alias: fi.app,
			Ref:   GetTargetRef(pvc.Name, apis.KindPersistentVolumeClaim),
		}
		bc.Spec.Task.Name = TaskPVCBackup
	})

	// transformFuncs provides a array of functions that made test specific change on the BackupConfiguration
	// apply these test specific changes
	for _, fn := range transformFuncs {
		fn(backupConfig)
	}

	By("Creating BackupConfiguration: " + backupConfig.Name)
	createdBC, err := fi.StashClient.StashV1beta1().BackupConfigurations(backupConfig.Namespace).Create(context.TODO(), backupConfig, metav1.CreateOptions{})
	fi.AppendToCleanupList(createdBC)

	By("Verifying that backup triggering CronJob has been created")
	fi.EventuallyCronJobCreated(backupConfig.ObjectMeta).Should(BeTrue())

	return createdBC, err
}

func (fi *Invocation) SetupRestoreProcessForPVC(pvc *core.PersistentVolumeClaim, repo *v1alpha1.Repository, transformFuncs ...func(restore *v1beta1.RestoreSession)) (*v1beta1.RestoreSession, error) {
	// Generate desired RestoreSession definition
	By("Creating RestoreSession")
	restoreSession := fi.GetRestoreSession(repo.Name, func(restore *v1beta1.RestoreSession) {
		restore.Spec.Target = &v1beta1.RestoreTarget{
			Alias: fi.app,
			Ref:   GetTargetRef(pvc.Name, apis.KindPersistentVolumeClaim),
			Rules: []v1beta1.Rule{
				{
					Snapshots: []string{"latest"},
				},
			},
		}
		restore.Spec.Task.Name = TaskPVCRestore
	})

	// transformFuncs provides a array of functions that made test specific change on the RestoreSession
	// apply these test specific changes.
	for _, fn := range transformFuncs {
		fn(restoreSession)
	}

	err := fi.CreateRestoreSession(restoreSession)
	fi.AppendToCleanupList(restoreSession)

	By("Waiting for restore process to complete")
	fi.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession).Should(BeTrue())

	return restoreSession, err
}

func (fi *Invocation) CleanupUndeletedPVCs() {
	pvcList, err := fi.KubeClient.CoreV1().PersistentVolumeClaims(fi.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		klog.Infoln(err)
		return
	}

	for _, pvc := range pvcList.Items {
		// cleanup only the pvc of this test
		if strings.Contains(pvc.Name, fi.app) {
			err = fi.KubeClient.CoreV1().PersistentVolumeClaims(fi.namespace).Delete(context.TODO(), pvc.Name, meta_util.DeleteInBackground())
			if err != nil {
				klog.Infoln(err)
			}
		}
	}
}
