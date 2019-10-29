package workloads

import (
	"fmt"

	"stash.appscode.dev/stash/apis"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("StatefulSet", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	var (
		setupRestoreProcessOnScaledUpSS = func(ss *apps.StatefulSet, repo *api.Repository) (*v1beta1.RestoreSession, error) {
			By("Creating RestoreSession")
			restoreSession := f.GetRestoreSessionForWorkload(repo.Name, framework.GetTargetRef(ss.Name, apis.KindStatefulSet))
			restoreSession.Spec.Rules = []v1beta1.Rule{
				{
					TargetHosts: []string{
						"host-3",
						"host-4",
					},
					SourceHost: "host-0",
					Paths: []string{
						framework.TestSourceDataMountPath,
					},
				},
				{
					TargetHosts: []string{},
					SourceHost:  "",
					Paths: []string{
						framework.TestSourceDataMountPath,
					},
				},
			}
			err := f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(restoreSession)

			By("Verifying that init-container has been injected")
			f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveInitContainer(util.StashInitContainer))

			By("Waiting for StatefulSet to be ready with init-container")
			err = f.WaitUntilStatefulSetWithInitContainer(ss.ObjectMeta)
			f.EventuallyPodAccessible(ss.ObjectMeta).Should(BeTrue())

			By("Waiting for restore process to complete")
			f.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta).Should(BeTrue())

			return restoreSession, err
		}
	)

	Context("StatefulSet", func() {

		Context("Restore in same StatefulSet", func() {
			It("should Backup & Restore in the source StatefulSet", func() {
				// Deploy a StatefulSet
				ss := f.DeployStatefulSet(fmt.Sprintf("source-ss1-%s", f.App()), int32(3))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(ss.ObjectMeta, repo, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source StatefulSet")
				err = f.CleanupSampleDataFromWorkload(ss.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data in the original StatefulSet")
				restoreSession, err := f.SetupRestoreProcess(ss.ObjectMeta, repo, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

				// Get restored data
				restoredData := f.RestoredData(ss.ObjectMeta, apis.KindStatefulSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Restore in different StatefulSet", func() {
			It("should restore backed up data into different StatefulSet", func() {
				// Deploy a StatefulSet
				ss := f.DeployStatefulSet(fmt.Sprintf("source-ss2-%s", f.App()), int32(3))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(ss.ObjectMeta, repo, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Deploy restored StatefulSet
				restoredSS := f.DeployStatefulSet(fmt.Sprintf("restored-ss2-%s", f.App()), int32(3))

				// Restore the backed up data
				By("Restoring the backed up data in the original StatefulSet")
				restoreSession, err := f.SetupRestoreProcess(restoredSS.ObjectMeta, repo, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

				// Get restored data
				restoredData := f.RestoredData(restoredSS.ObjectMeta, apis.KindStatefulSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Restore on scaled up StatefulSet", func() {
			It("should restore backed up data into scaled up StatefulSet", func() {
				// Deploy a StatefulSet
				ss := f.DeployStatefulSet(fmt.Sprintf("source-ss3-%s", f.App()), int32(3))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(ss.ObjectMeta, repo, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Deploy restored StatefulSet
				restoredSS := f.DeployStatefulSet(fmt.Sprintf("restored-ss3-%s", f.App()), int32(5))

				// Restore the backed up data
				By("Restoring the backed up data in different StatefulSet")
				restoreSession, err := setupRestoreProcessOnScaledUpSS(restoredSS, repo)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

				// Get restored data
				restoredData := f.RestoredData(restoredSS.ObjectMeta, apis.KindStatefulSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

	})

})
