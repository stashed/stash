package volumes

import (
	"fmt"

	"stash.appscode.dev/stash/apis"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Volume", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	var (
		setupPVCBackup = func(pvc *core.PersistentVolumeClaim, repo *api.Repository) (*v1beta1.BackupConfiguration, error) {
			// Generate desired BackupConfiguration definition
			backupConfig := f.GetBackupConfigurationForWorkload(repo.Name, framework.GetTargetRef(pvc.Name, apis.KindPersistentVolumeClaim))
			backupConfig.Spec.Target = f.PVCBackupTarget(pvc.Name)
			backupConfig.Spec.Task.Name = framework.TaskPVCBackup

			By("Creating BackupConfiguration: " + backupConfig.Name)
			createdBC, err := f.StashClient.StashV1beta1().BackupConfigurations(backupConfig.Namespace).Create(backupConfig)
			f.AppendToCleanupList(createdBC)

			By("Verifying that backup triggering CronJob has been created")
			f.EventuallyCronJobCreated(backupConfig.ObjectMeta).Should(BeTrue())

			return createdBC, err
		}

		setupRestoreProcessForPVC = func(pvc *core.PersistentVolumeClaim, repo *api.Repository) (*v1beta1.RestoreSession, error) {
			// Generate desired RestoreSession definition
			By("Creating RestoreSession")
			restoreSession := f.GetRestoreSessionForWorkload(repo.Name, framework.GetTargetRef(pvc.Name, apis.KindPersistentVolumeClaim))
			restoreSession.Spec.Target = f.PVCRestoreTarget(pvc.Name)
			restoreSession.Spec.Rules = []v1beta1.Rule{
				{
					Paths: []string{
						framework.TestSourceDataMountPath,
					},
				},
			}
			restoreSession.Spec.Task.Name = framework.TaskPVCRestore

			err := f.CreateRestoreSession(restoreSession)
			f.AppendToCleanupList(restoreSession)

			By("Waiting for restore process to complete")
			f.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta).Should(BeTrue())

			return restoreSession, err
		}
	)

	Context("PVC", func() {

		Context("Restore in same PVC", func() {
			It("should Backup & Restore in the source PVC", func() {
				// Create new PVC
				pvc := f.CreateNewPVC(fmt.Sprintf("source-pvc-%s", f.App()))
				// Deploy a Pod
				pod := f.DeployPod(pvc.Name)

				// Generate Sample Data
				sampleData := f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup PVC Backup
				backupConfig, err := setupPVCBackup(pvc, repo)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source Pod")
				err = f.CleanupSampleDataFromWorkload(pod.ObjectMeta, apis.KindPod)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data in the original Pod")
				restoreSession, err := setupRestoreProcessForPVC(pvc, repo)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

				// Get restored data
				restoredData := f.RestoredData(pod.ObjectMeta, apis.KindPod)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Restore in different PVC", func() {
			It("should restore backed up data into different PVC", func() {
				// Create new PVC
				pvc := f.CreateNewPVC(fmt.Sprintf("source-pvc1-%s", f.App()))
				// Deploy a Pod
				pod := f.DeployPod(pvc.Name)

				// Generate Sample Data
				sampleData := f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup PVC Backup
				backupConfig, err := setupPVCBackup(pvc, repo)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Create restored Pvc
				restoredPVC := f.CreateNewPVC(fmt.Sprintf("restore-pvc-%s", f.App()))

				// Deploy another Pod
				restoredPod := f.DeployPod(restoredPVC.Name)

				// Restore the backed up data
				By("Restoring the backed up data in the original Pod")
				restoreSession, err := setupRestoreProcessForPVC(restoredPVC, repo)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

				// Get restored data
				restoredData := f.RestoredData(restoredPod.ObjectMeta, apis.KindPod)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

	})

})
