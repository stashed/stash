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

var _ = Describe("Deployment", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Deployment", func() {

		Context("Restore in same Deployment", func() {
			It("should Backup & Restore in the source Deployment", func() {
				// Deploy a Deployment
				deployment := f.DeployDeployment(fmt.Sprintf("source-deployment1-%s", f.App()), int32(1))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source Deployment")
				err = f.CleanupSampleDataFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data in the original Deployment")
				restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

				// Get restored data
				restoredData := f.RestoredData(deployment.ObjectMeta, apis.KindDeployment)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Restore in different Deployment", func() {
			It("should restore backed up data into different Deployment", func() {
				// Deploy a Deployment
				deployment := f.DeployDeployment(fmt.Sprintf("source-deployment2-%s", f.App()), int32(1))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Deploy restored Deployment
				restoredDeployment := f.DeployDeployment(fmt.Sprintf("restored-deployment-%s", f.App()), int32(1))

				// Restore the backed up data
				By("Restoring the backed up data in different Deployment")
				restoreSession, err := f.SetupRestoreProcess(restoredDeployment.ObjectMeta, repo, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

				// Get restored data
				restoredData := f.RestoredData(restoredDeployment.ObjectMeta, apis.KindDeployment)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Leader election for backup and restore Deployment", func() {
			It("Should leader elect and backup and restore Deployment", func() {
				// Deploy a Deployment
				deployment := f.DeployDeployment(fmt.Sprintf("source-deployment3-%s", f.App()), int32(2))

				//  Generate Sample Data
				sampleData := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for leader election")
				f.CheckLeaderElection(deployment.ObjectMeta, apis.KindDeployment, v1beta1.ResourceKindBackupConfiguration)

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source Deployment")
				err = f.CleanupSampleDataFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data in the original Deployment")
				restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

				// Get restored data
				restoredData := f.RestoredData(deployment.ObjectMeta, apis.KindDeployment)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

	})

})
