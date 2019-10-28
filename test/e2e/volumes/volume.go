package volumes

import (
	"fmt"

	"stash.appscode.dev/stash/apis/stash/v1beta1"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	"github.com/appscode/go/sets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kmodules.xyz/client-go/core/v1"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
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
		deployPod = func(pvcName string) *core.Pod {
			// Generate Pod definition
			pod := f.Pod(pvcName)

			By("Deploying Pod: " + pod.Name)
			createdPod, err := f.CreatePod(pod)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdPod)

			By("Waiting for Pod to be ready")
			err = v1.WaitUntilPodRunning(f.KubeClient, createdPod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdPod.ObjectMeta).Should(BeTrue())

			return createdPod
		}

		setupPVCBackup = func(pvc *core.PersistentVolumeClaim, repo *api.Repository) *v1beta1.BackupConfiguration {
			// Generate desired BackupConfiguration definition
			backupConfig := f.GetBackupConfigurationForWorkload(repo.Name, framework.GetTargetRef(pvc.Name, apis.KindPersistentVolumeClaim))
			backupConfig.Spec.Target = f.PVCBackupTarget(pvc.Name)
			backupConfig.Spec.Task.Name = framework.TaskPVCBackup

			By("Creating BackupConfiguration: " + backupConfig.Name)
			createdBC, err := f.StashClient.StashV1beta1().BackupConfigurations(backupConfig.Namespace).Create(backupConfig)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdBC)

			By("Verifying that backup triggering CronJob has been created")
			f.EventuallyCronJobCreated(backupConfig.ObjectMeta).Should(BeTrue())

			return createdBC
		}

		restoreData = func(pvc *core.PersistentVolumeClaim, pod *core.Pod, repo *api.Repository) sets.String {
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
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(restoreSession)

			By("Waiting for restore process to complete")
			f.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta).Should(BeTrue())

			By("Verifying that RestoreSession succeeded")
			completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

			By("Reading restored data")
			restoredData, err := f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPod)
			Expect(err).NotTo(HaveOccurred())
			Expect(restoredData).NotTo(BeEmpty())

			return restoredData
		}
	)

	Context("PVC", func() {

		Context("Restore in same PVC", func() {
			It("should Backup & Restore in the source PVC", func() {
				// Create new PVC
				pvc := f.CreateNewPVC(fmt.Sprintf("source-pvc-%s", f.App()))
				// Deploy a Pod
				pod := deployPod(pvc.Name)

				// Generate Sample Data
				sampleData := f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := setupPVCBackup(pvc, repo)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source Pod")
				err = f.CleanupSampleDataFromWorkload(pod.ObjectMeta, apis.KindPod)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original Pod")
				restoredData := restoreData(pvc, pod, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Restore in different PVC", func() {
			It("should restore backed up data into different PVC", func() {
				// Create new PVC
				pvc := f.CreateNewPVC(fmt.Sprintf("source-diff-pvc-%s", f.App()))
				// Deploy a Pod
				pod := deployPod(pvc.Name)

				// Generate Sample Data
				sampleData := f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := setupPVCBackup(pvc, repo)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)

				// Create restored Pvc
				restoredPVC := f.CreateNewPVC(fmt.Sprintf("restore-pvc-%s", f.App()))

				// Deploy another Pod
				restoredPod := deployPod(restoredPVC.Name)

				// Restore the backup data
				By("Restoring the backed up data in the restored PVC")
				restoredData := restoreData(restoredPVC, restoredPod, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

	})

})
