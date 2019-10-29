package framework

import (
	"fmt"

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
)

func (f *Framework) GenerateSampleData(objMeta metav1.ObjectMeta, kind string) sets.String {
	By("Generating sample data inside workload pods")
	err := f.CreateSampleDataInsideWorkload(objMeta, kind)
	Expect(err).NotTo(HaveOccurred())

	By("Verifying that sample data has been generated")
	sampleData, err := f.ReadSampleDataFromFromWorkload(objMeta, kind)
	Expect(err).NotTo(HaveOccurred())
	Expect(sampleData).ShouldNot(BeEmpty())

	return sampleData
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

func (f *Invocation) TakeInstantBackup(objMeta metav1.ObjectMeta) (*v1beta1.BackupSession, error) {
	// Trigger Instant Backup
	By("Triggering Instant Backup")
	backupSession, err := f.TriggerInstantBackup(objMeta)
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

func (f Invocation) AddAutoBackupAnnotations(annotations map[string]string, obj interface{}) {
	By("Adding auto-backup specific annotations to the Workload")
	err := f.AddAutoBackupAnnotationsToTarget(annotations, obj)
	Expect(err).NotTo(HaveOccurred())

	By("Verifying that the auto-backup annotations has been added successfully")
	f.EventuallyAutoBackupAnnotationsFound(annotations, obj).Should(BeTrue())
}

func (f Invocation) VerifyAutoBackupConfigured(workloadMeta metav1.ObjectMeta, kind string) *v1beta1.BackupConfiguration {
	// BackupBlueprint create BackupConfiguration and Repository such that
	// the name of the BackupConfiguration and Repository will follow
	// the patter: <lower case of the workload kind>-<workload name>.
	// we will form the meta name and namespace for farther process.
	objMeta := metav1.ObjectMeta{
		Namespace: f.Namespace(),
	}
	switch kind {
	case apis.KindDeployment:
		objMeta.Name = fmt.Sprintf("deployment-%s", workloadMeta.Name)
	case apis.KindDaemonSet:
		objMeta.Name = fmt.Sprintf("daemonset-%s", workloadMeta.Name)
	case apis.KindStatefulSet:
		objMeta.Name = fmt.Sprintf("statefulset-%s", workloadMeta.Name)
	case apis.KindReplicationController:
		objMeta.Name = fmt.Sprintf("replicationcontroller-%s", workloadMeta.Name)
	case apis.KindReplicaSet:
		objMeta.Name = fmt.Sprintf("replicaset-%s", workloadMeta.Name)
	case apis.KindPersistentVolumeClaim:
		objMeta.Name = fmt.Sprintf("persistentvolumeclaim-%s", workloadMeta.Name)
	}

	By("Waiting for Repository")
	f.EventuallyRepositoryCreated(objMeta).Should(BeTrue())

	By("Waiting for BackupConfiguration")
	f.EventuallyBackupConfigurationCreated(objMeta).Should(BeTrue())
	backupConfig, err := f.StashClient.StashV1beta1().BackupConfigurations(objMeta.Namespace).Get(objMeta.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

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
	Expect(err).NotTo(HaveOccurred())

	return backupConfig
}
