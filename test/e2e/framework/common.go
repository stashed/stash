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
	"stash.appscode.dev/stash/apis"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	"github.com/appscode/go/sets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kmodules.xyz/client-go/meta"
)

func (f *Framework) GenerateSampleData(objMeta metav1.ObjectMeta, kind string) (sets.String, error) {
	By("Generating sample data inside workload pods")
	err := f.CreateSampleDataInsideWorkload(objMeta, kind)
	if err != nil {
		return sets.String{}, err
	}

	By("Verifying that sample data has been generated")
	sampleData, err := f.ReadSampleDataFromFromWorkload(objMeta, kind)
	Expect(err).NotTo(HaveOccurred())
	Expect(sampleData).ShouldNot(BeEmpty())

	return sampleData, nil
}

func (f *Invocation) SetupWorkloadBackup(objMeta metav1.ObjectMeta, repo *api.Repository, kind string) (*v1beta1.BackupConfiguration, error) {
	// Generate desired BackupConfiguration definition
	backupConfig := f.GetBackupConfigurationForWorkload(repo.Name, GetTargetRef(objMeta.Name, kind))

	By("Creating BackupConfiguration: " + backupConfig.Name)
	createdBC, err := f.StashClient.StashV1beta1().BackupConfigurations(backupConfig.Namespace).Create(backupConfig)
	Expect(err).NotTo(HaveOccurred())
	f.AppendToCleanupList(createdBC)

	By("Verifying that backup triggering CronJob has been created")
	f.EventuallyCronJobCreated(backupConfig.ObjectMeta).Should(BeTrue())

	By("Verifying that sidecar has been injected")
	switch kind {
	case apis.KindDeployment:
		f.EventuallyDeployment(objMeta).Should(HaveSidecar(util.StashContainer))
		By("Waiting for Deployment to be ready with sidecar")
		err = f.WaitUntilDeploymentReadyWithSidecar(objMeta)
	case apis.KindDaemonSet:
		f.EventuallyDaemonSet(objMeta).Should(HaveSidecar(util.StashContainer))
		By("Waiting for DaemonSet to be ready with sidecar")
		err = f.WaitUntilDaemonSetReadyWithSidecar(objMeta)
	case apis.KindStatefulSet:
		f.EventuallyStatefulSet(objMeta).Should(HaveSidecar(util.StashContainer))
		By("Waiting for StatefulSet to be ready with sidecar")
		err = f.WaitUntilStatefulSetReadyWithSidecar(objMeta)
	case apis.KindReplicaSet:
		f.EventuallyReplicaSet(objMeta).Should(HaveSidecar(util.StashContainer))
		By("Waiting for ReplicaSet to be ready with sidecar")
		err = f.WaitUntilRSReadyWithSidecar(objMeta)
	case apis.KindReplicationController:
		f.EventuallyReplicationController(objMeta).Should(HaveSidecar(util.StashContainer))
		By("Waiting for ReplicationController to be ready with sidecar")
		err = f.WaitUntilRCReadyWithSidecar(objMeta)
	}
	return createdBC, err
}

func (f *Invocation) SetupWorkloadBackupForBackupBatch(targetRefs []v1beta1.TargetRef, repo *api.Repository, transformFunc func(in *v1beta1.BackupBatch)) (*v1beta1.BackupBatch, error) {
	// Generate desired BackupBatch definition
	backupBatch := f.BackupBatch(repo.Name)
	// transformFunc provide a function that made test specific change on the backupBatch
	// apply these test specific changes
	transformFunc(backupBatch)

	By("Creating BackupBatch: " + backupBatch.Name)
	createdBackupBatch, err := f.StashClient.StashV1beta1().BackupBatches(backupBatch.Namespace).Create(backupBatch)
	Expect(err).NotTo(HaveOccurred())
	f.AppendToCleanupList(createdBackupBatch)

	By("Verifying that backup triggering CronJob has been created")
	f.EventuallyCronJobCreated(backupBatch.ObjectMeta).Should(BeTrue())

	By("Verifying that sidecar has been injected")
	for _, backupConfigTemp := range backupBatch.Spec.BackupConfigurationTemplates {
		objMeta := metav1.ObjectMeta{
			Namespace: backupBatch.Namespace,
			Name:      backupConfigTemp.Spec.Target.Ref.Name,
		}
		switch backupConfigTemp.Spec.Target.Ref.Kind {
		case apis.KindDeployment:
			f.EventuallyDeployment(objMeta).Should(HaveSidecar(util.StashContainer))
			By("Waiting for Deployment to be ready with sidecar")
			err = f.WaitUntilDeploymentReadyWithSidecar(objMeta)
		case apis.KindDaemonSet:
			f.EventuallyDaemonSet(objMeta).Should(HaveSidecar(util.StashContainer))
			By("Waiting for DaemonSet to be ready with sidecar")
			err = f.WaitUntilDaemonSetReadyWithSidecar(objMeta)
		case apis.KindStatefulSet:
			f.EventuallyStatefulSet(objMeta).Should(HaveSidecar(util.StashContainer))
			By("Waiting for StatefulSet to be ready with sidecar")
			err = f.WaitUntilStatefulSetReadyWithSidecar(objMeta)
		case apis.KindReplicaSet:
			f.EventuallyReplicaSet(objMeta).Should(HaveSidecar(util.StashContainer))
			By("Waiting for ReplicaSet to be ready with sidecar")
			err = f.WaitUntilRSReadyWithSidecar(objMeta)
		case apis.KindReplicationController:
			f.EventuallyReplicationController(objMeta).Should(HaveSidecar(util.StashContainer))
			By("Waiting for ReplicationController to be ready with sidecar")
			err = f.WaitUntilRCReadyWithSidecar(objMeta)
		}
		if err != nil {
			return createdBackupBatch, err
		}
	}
	return createdBackupBatch, err
}

func (f *Invocation) TakeInstantBackup(objMeta metav1.ObjectMeta, targetRef v1beta1.TargetRef) (*v1beta1.BackupSession, error) {
	// Trigger Instant Backup
	By("Triggering Instant Backup")
	backupSession, err := f.TriggerInstantBackup(objMeta, targetRef)
	if err != nil {
		return backupSession, err
	}
	f.AppendToCleanupList(backupSession)

	By("Waiting for backup process to complete")
	f.EventuallyBackupProcessCompleted(backupSession.ObjectMeta).Should(BeTrue())

	return backupSession, nil
}

func (f *Invocation) RestoredData(objMeta metav1.ObjectMeta, kind string) sets.String {
	By("Reading restored data")
	restoredData, err := f.ReadSampleDataFromFromWorkload(objMeta, kind)
	Expect(err).NotTo(HaveOccurred())
	Expect(restoredData).NotTo(BeEmpty())

	return restoredData
}

func (f *Invocation) SetupRestoreProcess(objMeta metav1.ObjectMeta, repo *api.Repository, kind string) (*v1beta1.RestoreSession, error) {
	By("Creating RestoreSession")
	restoreSession := f.GetRestoreSessionForWorkload(repo.Name, GetTargetRef(objMeta.Name, kind))
	err := f.CreateRestoreSession(restoreSession)
	Expect(err).NotTo(HaveOccurred())
	f.AppendToCleanupList(restoreSession)

	By("Verifying that init-container has been injected")
	switch kind {
	case apis.KindDeployment:
		f.EventuallyDeployment(objMeta).Should(HaveInitContainer(util.StashInitContainer))
		By("Waiting for workload to be ready with init-container")
		err = f.WaitUntilDeploymentReadyWithInitContainer(objMeta)
	case apis.KindDaemonSet:
		f.EventuallyDaemonSet(objMeta).Should(HaveInitContainer(util.StashInitContainer))
		By("Waiting for workload to be ready with init-container")
		err = f.WaitUntilDaemonSetReadyWithInitContainer(objMeta)
	case apis.KindStatefulSet:
		f.EventuallyStatefulSet(objMeta).Should(HaveInitContainer(util.StashInitContainer))
		By("Waiting for workload to be ready with init-container")
		err = f.WaitUntilStatefulSetWithInitContainer(objMeta)
	case apis.KindReplicaSet:
		f.EventuallyReplicaSet(objMeta).Should(HaveInitContainer(util.StashInitContainer))
		By("Waiting for workload to be ready with init-container")
		err = f.WaitUntilRSReadyWithInitContainer(objMeta)
	case apis.KindReplicationController:
		f.EventuallyReplicationController(objMeta).Should(HaveInitContainer(util.StashInitContainer))
		By("Waiting for workload to be ready with init-container")
		err = f.WaitUntilRCReadyWithInitContainer(objMeta)
	}
	f.EventuallyPodAccessible(objMeta).Should(BeTrue())
	By("Waiting for restore process to complete")
	f.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta).Should(BeTrue())

	return restoreSession, err
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
	}
	return targetRef
}

func (f Invocation) AddAutoBackupAnnotations(annotations map[string]string, obj interface{}) error {
	By("Adding auto-backup specific annotations to the Target")
	err := f.AddAnnotations(annotations, obj)
	if err != nil {
		return err
	}

	By("Verifying that the auto-backup annotations has been added successfully")
	f.EventuallyAnnotationsFound(annotations, obj).Should(BeTrue())
	return nil
}

func (f Invocation) VerifyAutoBackupConfigured(workloadMeta metav1.ObjectMeta, kind string) (*v1beta1.BackupConfiguration, error) {
	// BackupBlueprint create BackupConfiguration and Repository such that
	// the name of the BackupConfiguration and Repository will follow
	// the patter: <lower case of the workload kind>-<workload name>.
	// we will form the meta name and namespace for farther process.
	objMeta := metav1.ObjectMeta{
		Namespace: f.Namespace(),
		Name:      meta.ValidNameWithPrefix(util.ResourceKindShortForm(kind), workloadMeta.Name),
	}

	By("Waiting for Repository")
	f.EventuallyRepositoryCreated(objMeta).Should(BeTrue())

	By("Waiting for BackupConfiguration")
	f.EventuallyBackupConfigurationCreated(objMeta).Should(BeTrue())
	backupConfig, err := f.StashClient.StashV1beta1().BackupConfigurations(objMeta.Namespace).Get(objMeta.Name, metav1.GetOptions{})
	if err != nil {
		return backupConfig, err
	}

	By("Verifying that backup triggering CronJob has been created")
	f.EventuallyCronJobCreated(objMeta).Should(BeTrue())

	By("Verifying that sidecar has been injected")
	switch kind {
	case apis.KindDeployment:
		f.EventuallyDeployment(workloadMeta).Should(HaveSidecar(util.StashContainer))
		By("Waiting for Deployment to be ready with sidecar")
		err = f.WaitUntilDeploymentReadyWithSidecar(workloadMeta)
	case apis.KindDaemonSet:
		f.EventuallyDaemonSet(workloadMeta).Should(HaveSidecar(util.StashContainer))
		By("Waiting for DaemonSet to be ready with sidecar")
		err = f.WaitUntilDaemonSetReadyWithSidecar(workloadMeta)
	case apis.KindStatefulSet:
		f.EventuallyStatefulSet(workloadMeta).Should(HaveSidecar(util.StashContainer))
		By("Waiting for StatefulSet to be ready with sidecar")
		err = f.WaitUntilStatefulSetReadyWithSidecar(workloadMeta)
	case apis.KindReplicaSet:
		f.EventuallyReplicaSet(workloadMeta).Should(HaveSidecar(util.StashContainer))
		By("Waiting for ReplicaSet to be ready with sidecar")
		err = f.WaitUntilRSReadyWithSidecar(workloadMeta)
	case apis.KindReplicationController:
		f.EventuallyReplicationController(workloadMeta).Should(HaveSidecar(util.StashContainer))
		By("Waiting for ReplicationController to be ready with sidecar")
		err = f.WaitUntilRCReadyWithSidecar(workloadMeta)
	}

	return backupConfig, err
}
