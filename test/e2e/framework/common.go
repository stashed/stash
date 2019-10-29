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
	store "kmodules.xyz/objectstore-api/api/v1"
)

func (f *Invocation) CreateNewPVC(name string) *core.PersistentVolumeClaim {
	// Generate PVC definition
	pvc := f.PersistentVolumeClaim()
	pvc.Name = name

	By(fmt.Sprintf("Creating PVC: %s/%s", pvc.Namespace, pvc.Name))
	createdPVC, err := f.CreatePersistentVolumeClaim(pvc)
	Expect(err).NotTo(HaveOccurred())
	f.AppendToCleanupList(createdPVC)

	return createdPVC
}

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

func (f *Invocation) SetupWorkloadBackup(objMeta metav1.ObjectMeta, repo *api.Repository, kind string) *v1beta1.BackupConfiguration {
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
	Expect(err).NotTo(HaveOccurred())

	return createdBC
}

func (f *Invocation) TakeInstantBackup(objMeta metav1.ObjectMeta) {
	// Trigger Instant Backup
	By("Triggering Instant Backup")
	backupSession, err := f.TriggerInstantBackup(objMeta)
	Expect(err).NotTo(HaveOccurred())
	f.AppendToCleanupList(backupSession)

	By("Waiting for backup process to complete")
	f.EventuallyBackupProcessCompleted(backupSession.ObjectMeta).Should(BeTrue())

	By("Verifying that BackupSession has succeeded")
	completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

}

func (f *Invocation) InstantBackupFailed(objMeta metav1.ObjectMeta) {
	// Trigger Instant Backup
	By("Triggering Instant Backup")
	backupSession, err := f.TriggerInstantBackup(objMeta)
	Expect(err).NotTo(HaveOccurred())
	f.AppendToCleanupList(backupSession)

	By("Waiting for backup process to complete")
	f.EventuallyBackupProcessCompleted(backupSession.ObjectMeta).Should(BeTrue())

	By("Verifying that BackupSession has failed")
	completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))

}

func (f *Invocation) RestoreData(objMeta metav1.ObjectMeta, repo *api.Repository, kind string) sets.String {
	By("Creating RestoreSession")
	restoreSession := f.GetRestoreSessionForWorkload(repo.Name, GetTargetRef(objMeta.Name, kind))
	err := f.CreateRestoreSession(restoreSession)
	Expect(err).NotTo(HaveOccurred())
	f.AppendToCleanupList(restoreSession)

	By("Verifying that init-container has been injected")
	switch kind {
	case apis.KindDeployment:
		f.EventuallyDeployment(objMeta).Should(HaveInitContainer(util.StashInitContainer))
	case apis.KindDaemonSet:
		f.EventuallyDaemonSet(objMeta).Should(HaveInitContainer(util.StashInitContainer))
	case apis.KindStatefulSet:
		f.EventuallyStatefulSet(objMeta).Should(HaveInitContainer(util.StashInitContainer))
	case apis.KindReplicaSet:
		f.EventuallyReplicaSet(objMeta).Should(HaveInitContainer(util.StashInitContainer))
	case apis.KindReplicationController:
		f.EventuallyReplicationController(objMeta).Should(HaveInitContainer(util.StashInitContainer))
	}

	By("Waiting for restore process to complete")
	f.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta).Should(BeTrue())

	By("Verifying that RestoreSession succeeded")
	completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

	By("Waiting for workload to be ready with init-container")
	switch kind {
	case apis.KindDeployment:
		err = f.WaitUntilDeploymentReadyWithInitContainer(objMeta)
	case apis.KindDaemonSet:
		err = f.WaitUntilDaemonSetReadyWithInitContainer(objMeta)
	case apis.KindStatefulSet:
		err = f.WaitUntilStatefulSetWithInitContainer(objMeta)
	case apis.KindReplicaSet:
		err = f.WaitUntilRSReadyWithInitContainer(objMeta)
	case apis.KindReplicationController:
		err = f.WaitUntilRCReadyWithInitContainer(objMeta)
	}
	Expect(err).NotTo(HaveOccurred())
	f.EventuallyPodAccessible(objMeta).Should(BeTrue())

	By("Reading restored data")
	restoredData, err := f.ReadSampleDataFromFromWorkload(objMeta, kind)
	Expect(err).NotTo(HaveOccurred())
	Expect(restoredData).NotTo(BeEmpty())

	return restoredData
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

func (f Invocation) AddAnnotationsToTarget(annotations map[string]string, obj interface{}) {
	By("Adding auto-backup specific annotations to the Workload")
	err := f.AddAutoBackupAnnotationsToTarget(annotations, obj)
	Expect(err).NotTo(HaveOccurred())

	By("Verifying that the auto-backup annotations has been added successfully")
	f.EventuallyAutoBackupAnnotationsFound(annotations, obj).Should(BeTrue())
}

func (f Invocation) CheckRepositoryAndBackupConfiguration(workloadMeta metav1.ObjectMeta, kind string) *v1beta1.BackupConfiguration {
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

func (f Invocation) CreateBackendSecretForMinio() *core.Secret {
	// Create Storage Secret
	cred := f.SecretForMinioBackend(true)

	if missing, _ := BeZero().Match(cred); missing {
		Skip("Missing Minio credential")
	}
	By(fmt.Sprintf("Creating Storage Secret for Minio: %s/%s", cred.Namespace, cred.Name))
	createdCred, err := f.CreateSecret(cred)
	Expect(err).NotTo(HaveOccurred())
	f.AppendToCleanupList(&cred)

	return createdCred
}

func (f Invocation) CreateNewBackupBlueprint(name string) *v1beta1.BackupBlueprint {
	// Create Secret for BackupBlueprint
	secret := f.CreateBackendSecretForMinio()

	// Generate BackupBlueprint definition
	bb := f.BackupBlueprint(f.GetRepositoryInfo(secret.Name))
	bb.Name = name

	By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
	createdBB, err := f.CreateBackupBlueprint(bb)
	Expect(err).NotTo(HaveOccurred())
	f.AppendToCleanupList(createdBB)
	return createdBB
}

func (f Invocation) GetRepositoryInfo(secretName string) api.RepositorySpec {
	repoInfo := api.RepositorySpec{
		Backend: store.Backend{
			S3: &store.S3Spec{
				Endpoint: f.MinioServiceAddres(),
				Bucket:   "minio-bucket",
				Prefix:   fmt.Sprintf("stash-e2e/%s/%s", f.Namespace(), f.App()),
			},
			StorageSecretName: secretName,
		},
		WipeOut: false,
	}
	return repoInfo
}
