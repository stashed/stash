/*
Copyright AppsCode Inc. and Contributors

Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"context"
	"path/filepath"

	"stash.appscode.dev/apimachinery/apis"
	api "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1alpha1 "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appcatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
)

func (fi *Invocation) GenerateSampleData(objMeta metav1.ObjectMeta, kind string) ([]string, error) {
	By("Generating sample data inside workload pods")
	err := fi.CreateSampleDataInsideWorkload(objMeta, kind)
	if err != nil {
		return nil, err
	}

	By("Verifying that sample data has been generated")
	sampleData, err := fi.ReadSampleDataFromFromWorkload(objMeta, kind)
	Expect(err).NotTo(HaveOccurred())
	Expect(sampleData).ShouldNot(BeEmpty())

	return sampleData, nil
}

func (fi *Invocation) GenerateBigSampleFile(meta metav1.ObjectMeta, kind string) ([]string, error) {
	By("Generating sample data inside workload pods")
	pod, err := fi.GetPod(meta)
	if err != nil {
		return nil, err
	}
	_, err = fi.ExecOnPod(pod, "truncate", "-s", "128M", filepath.Join(TestSourceDataMountPath, "file.txt"))
	if err != nil {
		return nil, err
	}

	By("Verifying that sample data has been generated")
	sampleData, err := fi.ReadSampleDataFromFromWorkload(meta, kind)
	Expect(err).NotTo(HaveOccurred())
	Expect(sampleData).ShouldNot(BeEmpty())

	return sampleData, nil
}

func (fi *Invocation) SetupWorkloadBackup(objMeta metav1.ObjectMeta, repo *api_v1alpha1.Repository, kind string, transformFuncs ...func(bc *v1beta1.BackupConfiguration)) (*v1beta1.BackupConfiguration, error) {

	backupConfig, err := fi.CreateBackupConfigForWorkload(objMeta, repo, kind, transformFuncs...)
	Expect(err).NotTo(HaveOccurred())

	By("Verifying that backup triggering CronJob has been created")
	fi.EventuallyCronJobCreated(backupConfig.ObjectMeta).Should(BeTrue())

	By("Verifying that sidecar has been injected")
	switch kind {
	case apis.KindDeployment:
		fi.EventuallyDeployment(objMeta).Should(HaveSidecar(apis.StashContainer))
		By("Waiting for Deployment to be ready with sidecar")
		err = fi.WaitUntilDeploymentReadyWithSidecar(objMeta)
	case apis.KindDaemonSet:
		fi.EventuallyDaemonSet(objMeta).Should(HaveSidecar(apis.StashContainer))
		By("Waiting for DaemonSet to be ready with sidecar")
		err = fi.WaitUntilDaemonSetReadyWithSidecar(objMeta)
	case apis.KindStatefulSet:
		fi.EventuallyStatefulSet(objMeta).Should(HaveSidecar(apis.StashContainer))
		By("Waiting for StatefulSet to be ready with sidecar")
		err = fi.WaitUntilStatefulSetReadyWithSidecar(objMeta)
	case apis.KindReplicaSet:
		fi.EventuallyReplicaSet(objMeta).Should(HaveSidecar(apis.StashContainer))
		By("Waiting for ReplicaSet to be ready with sidecar")
		err = fi.WaitUntilRSReadyWithSidecar(objMeta)
	case apis.KindReplicationController:
		fi.EventuallyReplicationController(objMeta).Should(HaveSidecar(apis.StashContainer))
		By("Waiting for ReplicationController to be ready with sidecar")
		err = fi.WaitUntilRCReadyWithSidecar(objMeta)
	}
	return backupConfig, err
}

func (fi *Invocation) CreateBackupConfigForWorkload(objMeta metav1.ObjectMeta, repo *api_v1alpha1.Repository, kind string, transformFuncs ...func(bc *v1beta1.BackupConfiguration)) (*v1beta1.BackupConfiguration, error) {
	// Generate desired BackupConfiguration definition
	backupConfig := fi.GetBackupConfiguration(repo.Name, func(bc *v1beta1.BackupConfiguration) {
		bc.Spec.Target = &v1beta1.BackupTarget{
			Alias: fi.app,
			Ref:   GetTargetRef(objMeta.Name, kind),
			Paths: []string{
				TestSourceDataMountPath,
			},
			VolumeMounts: []core.VolumeMount{
				{
					Name:      SourceVolume,
					MountPath: TestSourceDataMountPath,
				},
			},
		}
	})
	// transformFuncs provides a array of functions that made test specific change on the BackupConfiguration
	// apply these test specific changes
	for _, fn := range transformFuncs {
		fn(backupConfig)
	}

	By("Creating BackupConfiguration: " + backupConfig.Name)
	createdBC, err := fi.StashClient.StashV1beta1().BackupConfigurations(backupConfig.Namespace).Create(context.TODO(), backupConfig, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	fi.AppendToCleanupList(createdBC)

	return createdBC, nil
}

func (fi *Invocation) WaitForTargetToBeReadyWithSidecar(kind string, objMeta metav1.ObjectMeta) error {
	switch kind {
	case apis.KindDeployment:
		By("Verifying that sidecar has been injected")
		fi.EventuallyDeployment(objMeta).Should(HaveSidecar(apis.StashContainer))
		By("Waiting for Deployment to be ready with sidecar")
		return fi.WaitUntilDeploymentReadyWithSidecar(objMeta)
	case apis.KindDaemonSet:
		By("Verifying that sidecar has been injected")
		fi.EventuallyDaemonSet(objMeta).Should(HaveSidecar(apis.StashContainer))
		By("Waiting for DaemonSet to be ready with sidecar")
		return fi.WaitUntilDaemonSetReadyWithSidecar(objMeta)
	case apis.KindStatefulSet:
		By("Verifying that sidecar has been injected")
		fi.EventuallyStatefulSet(objMeta).Should(HaveSidecar(apis.StashContainer))
		By("Waiting for StatefulSet to be ready with sidecar")
		return fi.WaitUntilStatefulSetReadyWithSidecar(objMeta)
	case apis.KindReplicaSet:
		By("Verifying that sidecar has been injected")
		fi.EventuallyReplicaSet(objMeta).Should(HaveSidecar(apis.StashContainer))
		By("Waiting for ReplicaSet to be ready with sidecar")
		return fi.WaitUntilRSReadyWithSidecar(objMeta)
	case apis.KindReplicationController:
		By("Verifying that sidecar has been injected")
		fi.EventuallyReplicationController(objMeta).Should(HaveSidecar(apis.StashContainer))
		By("Waiting for ReplicationController to be ready with sidecar")
		return fi.WaitUntilRCReadyWithSidecar(objMeta)
	}
	return nil
}

func (fi *Invocation) TakeInstantBackup(objMeta metav1.ObjectMeta, invokerRef v1beta1.BackupInvokerRef) (*v1beta1.BackupSession, error) {
	// Trigger Instant Backup
	By("Triggering Instant Backup")
	backupSession, err := fi.TriggerInstantBackup(objMeta, invokerRef)
	if err != nil {
		return backupSession, err
	}
	fi.AppendToCleanupList(backupSession)

	By("Waiting for backup process to complete")
	fi.EventuallyBackupProcessCompleted(backupSession.ObjectMeta).Should(BeTrue())

	return backupSession, nil
}

func (fi *Invocation) RestoredData(objMeta metav1.ObjectMeta, kind string) []string {
	fi.EventuallyAllPodsAccessible(objMeta).Should(BeTrue())
	By("Reading restored data")
	restoredData, err := fi.ReadSampleDataFromFromWorkload(objMeta, kind)
	Expect(err).NotTo(HaveOccurred())
	Expect(restoredData).NotTo(BeEmpty())

	return restoredData
}

func (fi *Invocation) SetupRestoreProcess(objMeta metav1.ObjectMeta, repo *api.Repository, kind, volumeName string, transformFuncs ...func(restore *v1beta1.RestoreSession)) (*v1beta1.RestoreSession, error) {
	// Create RestoreSession
	restoreSession, err := fi.CreateRestoreSessionForWorkload(objMeta, repo.Name, kind, volumeName, transformFuncs...)
	if err != nil {
		return nil, err
	}
	By("Verifying that init-container has been injected")
	switch kind {
	case apis.KindDeployment:
		fi.EventuallyDeployment(objMeta).Should(HaveInitContainer(apis.StashInitContainer))
	case apis.KindDaemonSet:
		fi.EventuallyDaemonSet(objMeta).Should(HaveInitContainer(apis.StashInitContainer))
	case apis.KindStatefulSet:
		fi.EventuallyStatefulSet(objMeta).Should(HaveInitContainer(apis.StashInitContainer))
	case apis.KindReplicaSet:
		fi.EventuallyReplicaSet(objMeta).Should(HaveInitContainer(apis.StashInitContainer))
	case apis.KindReplicationController:
		fi.EventuallyReplicationController(objMeta).Should(HaveInitContainer(apis.StashInitContainer))
	}

	By("Waiting for restore process to complete")
	fi.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession).Should(BeTrue())

	return restoreSession, err
}

func (fi *Invocation) CreateRestoreSessionForWorkload(objMeta metav1.ObjectMeta, repoName, kind, volumeName string, transformFuncs ...func(restore *v1beta1.RestoreSession)) (*v1beta1.RestoreSession, error) {
	// Generate desired BackupConfiguration definition
	restoreSession := fi.GetRestoreSession(repoName, func(restore *v1beta1.RestoreSession) {
		restore.Spec.Target = &v1beta1.RestoreTarget{
			Alias: fi.app,
			Rules: []v1beta1.Rule{
				{
					Paths: []string{TestSourceDataMountPath},
				},
			},
			VolumeMounts: []core.VolumeMount{
				{
					Name:      volumeName,
					MountPath: TestSourceDataMountPath,
				},
			},
		}
	})
	// if objMeta is provided, then set target reference
	if objMeta.Name != "" {
		restoreSession.Spec.Target.Ref = GetTargetRef(objMeta.Name, kind)
	}
	// transformFuncs provides a array of functions that made test specific change on the RestoreSession
	// apply these test specific changes.
	for _, fn := range transformFuncs {
		fn(restoreSession)
	}

	By("Creating RestoreSession")
	err := fi.CreateRestoreSession(restoreSession)
	Expect(err).NotTo(HaveOccurred())
	fi.AppendToCleanupList(restoreSession)

	return restoreSession, nil
}

func GetTargetRef(name string, kind string) v1beta1.TargetRef {
	targetRef := v1beta1.TargetRef{
		Name: name,
	}
	switch kind {
	case apis.KindDeployment:
		targetRef.Kind = apis.KindDeployment
		targetRef.APIVersion = apps.SchemeGroupVersion.String()
	case apis.KindDaemonSet:
		targetRef.Kind = apis.KindDaemonSet
		targetRef.APIVersion = apps.SchemeGroupVersion.String()
	case apis.KindStatefulSet:
		targetRef.Kind = apis.KindStatefulSet
		targetRef.APIVersion = apps.SchemeGroupVersion.String()
	case apis.KindReplicationController:
		targetRef.Kind = apis.KindReplicationController
		targetRef.APIVersion = core.SchemeGroupVersion.String()
	case apis.KindReplicaSet:
		targetRef.Kind = apis.KindReplicaSet
		targetRef.APIVersion = apps.SchemeGroupVersion.String()
	case apis.KindPersistentVolumeClaim:
		targetRef.Kind = apis.KindPersistentVolumeClaim
		targetRef.APIVersion = core.SchemeGroupVersion.String()
	case apis.KindAppBinding:
		targetRef.Kind = apis.KindAppBinding
		targetRef.APIVersion = appcatalog.SchemeGroupVersion.String()
	}
	return targetRef
}

func (fi *Invocation) PrintDebugInfoOnFailure() {
	if CurrentGinkgoTestDescription().Failed {
		fi.PrintDebugHelpers()
		TestFailed = true
	}
}
