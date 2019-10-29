package workloads

import (
	"fmt"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ReplicationController", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("ReplicationController", func() {

		Context("Restore in same ReplicationController", func() {
			It("should Backup & Restore in the source ReplicationController", func() {
				// Deploy a ReplicationController
				rc := f.DeployReplicationController(fmt.Sprintf("source-rc1-%s", f.App()), int32(1))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(rc.ObjectMeta, apis.KindReplicationController)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(rc.ObjectMeta, repo, apis.KindReplicationController)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source ReplicationController")
				err = f.CleanupSampleDataFromWorkload(rc.ObjectMeta, apis.KindReplicationController)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data in the original ReplicationController")
				restoreSession, err := f.SetupRestoreProcess(rc.ObjectMeta, repo, apis.KindReplicationController)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

				// Get restored data
				restoredData := f.RestoredData(rc.ObjectMeta, apis.KindReplicationController)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Restore in different ReplicationController", func() {
			It("should restore backed up data into different ReplicationController", func() {
				// Deploy a ReplicationController
				rc := f.DeployReplicationController(fmt.Sprintf("source-rc2-%s", f.App()), int32(1))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(rc.ObjectMeta, apis.KindReplicationController)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(rc.ObjectMeta, repo, apis.KindReplicationController)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Deploy restored ReplicationController
				restoredRC := f.DeployReplicationController(fmt.Sprintf("restored-rc-%s", f.App()), int32(1))

				// Restore the backed up data
				By("Restoring the backed up data in different ReplicationController")
				restoreSession, err := f.SetupRestoreProcess(restoredRC.ObjectMeta, repo, apis.KindReplicationController)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

				// Get restored data
				restoredData := f.RestoredData(restoredRC.ObjectMeta, apis.KindReplicationController)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Leader election for backup and restore ReplicationController", func() {
			It("Should leader elect and backup and restore ReplicationController", func() {
				// Deploy a ReplicationController
				rc := f.DeployReplicationController(fmt.Sprintf("source-rc3-%s", f.App()), int32(2))

				//  Generate Sample Data
				sampleData := f.GenerateSampleData(rc.ObjectMeta, apis.KindReplicationController)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(rc.ObjectMeta, repo, apis.KindReplicationController)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for leader election")
				f.CheckLeaderElection(rc.ObjectMeta, apis.KindReplicationController, v1beta1.ResourceKindBackupConfiguration)

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source ReplicationController")
				err = f.CleanupSampleDataFromWorkload(rc.ObjectMeta, apis.KindReplicationController)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data in the original ReplicationController")
				restoreSession, err := f.SetupRestoreProcess(rc.ObjectMeta, repo, apis.KindReplicationController)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

				// Get restored data
				restoredData := f.RestoredData(rc.ObjectMeta, apis.KindReplicationController)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

	})

})
