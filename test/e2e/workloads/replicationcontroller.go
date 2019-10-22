package workloads

import (
	"fmt"
	"strings"

	"stash.appscode.dev/stash/apis"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	"github.com/appscode/go/sets"
	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
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

	var (
		createPVC = func(name string) *core.PersistentVolumeClaim {
			// Generate PVC definition
			pvc := f.PersistentVolumeClaim()
			pvc.Name = fmt.Sprintf("%s-pvc-%s", strings.Split(name, "-")[0], f.App())

			By("Creating PVC: " + pvc.Name)
			createdPVC, err := f.CreatePersistentVolumeClaim(pvc)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdPVC)

			return createdPVC
		}

		deployRC = func(name string) *core.ReplicationController {
			// Create PVC for ReplicationController
			pvc := createPVC(name)
			// Generate ReplicationController definition
			rc := f.ReplicationController(pvc.Name)
			rc.Name = name

			By("Deploying ReplicationController: " + rc.Name)
			createdRC, err := f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdRC)

			By("Waiting for ReplicationController to be ready")
			err = util.WaitUntilRCReady(f.KubeClient, createdRC.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdRC.ObjectMeta).Should(BeTrue())

			return createdRC
		}

		deployRCWithMultipleReplica = func(name string) *core.ReplicationController {
			// Create PVC for ReplicationController
			pvc := createPVC(fmt.Sprintf("source-pvc-%s", f.App()))
			// Generate ReplicationController definition
			rc := f.ReplicationController(pvc.Name)
			rc.Spec.Replicas = types.Int32P(2) // two replicas
			rc.Name = name

			By("Deploying ReplicationController: " + rc.Name)
			createdRC, err := f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdRC)

			By("Waiting for ReplicationController to be ready")
			err = util.WaitUntilRCReady(f.KubeClient, createdRC.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdRC.ObjectMeta).Should(BeTrue())

			return createdRC
		}

		generateSampleData = func(rc *core.ReplicationController) sets.String {
			By("Generating sample data inside ReplicationController pods")
			err := f.CreateSampleDataInsideWorkload(rc.ObjectMeta, apis.KindReplicationController)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that sample data has been generated")
			sampleData, err := f.ReadSampleDataFromFromWorkload(rc.ObjectMeta, apis.KindReplicationController)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleData).ShouldNot(BeEmpty())

			return sampleData
		}

		getTargetRef = func(rc *core.ReplicationController) v1beta1.TargetRef {
			return v1beta1.TargetRef{
				Name:       rc.Name,
				Kind:       apis.KindReplicationController,
				APIVersion: "v1",
			}
		}

		setupRCBackup = func(rc *core.ReplicationController, repo *api.Repository) *v1beta1.BackupConfiguration {
			// Generate desired BackupConfiguration definition
			backupConfig := f.GetBackupConfigurationForWorkload(repo.Name, getTargetRef(rc))

			By("Creating BackupConfiguration: " + backupConfig.Name)
			createdBC, err := f.StashClient.StashV1beta1().BackupConfigurations(backupConfig.Namespace).Create(backupConfig)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdBC)

			By("Verifying that backup triggering CronJob has been created")
			f.EventuallyCronJobCreated(backupConfig.ObjectMeta).Should(BeTrue())

			By("Verifying that sidecar has been injected")
			f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for ReplicationController to be ready with sidecar")
			err = f.WaitUntilRCReadyWithSidecar(rc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			return createdBC
		}

		takeInstantBackup = func(rc *core.ReplicationController, repo *api.Repository) {
			// Setup Backup
			backupConfig := setupRCBackup(rc, repo)

			// Trigger Instant Backup
			By("Triggering Instant Backup")
			backupSession, err := f.TriggerInstantBackup(backupConfig)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(backupSession)

			By("Waiting for backup process to complete")
			f.EventuallyBackupProcessCompleted(backupSession.ObjectMeta).Should(BeTrue())

			By("Verifying that BackupSession has succeeded")
			completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
		}

		restoreData = func(rc *core.ReplicationController, repo *api.Repository) sets.String {
			By("Creating RestoreSession")
			restoreSession := f.GetRestoreSessionForWorkload(repo.Name, getTargetRef(rc))
			err := f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(restoreSession)

			By("Verifying that init-container has been injected")
			f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveInitContainer(util.StashInitContainer))

			By("Waiting for restore process to complete")
			f.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta).Should(BeTrue())

			By("Verifying that RestoreSession succeeded")
			completedRestore, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRestore.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

			By("Waiting for ReplicationController to be ready with init-container")
			err = f.WaitUntilRCReadyWithInitContainer(rc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			f.EventuallyPodAccessible(rc.ObjectMeta).Should(BeTrue())

			By("Reading restored data")
			restoredData, err := f.ReadSampleDataFromFromWorkload(rc.ObjectMeta, apis.KindReplicationController)
			Expect(err).NotTo(HaveOccurred())
			Expect(restoredData).NotTo(BeEmpty())

			return restoredData
		}
	)

	Context("ReplicationController", func() {

		Context("Restore in same ReplicationController", func() {

			It("should Backup & Restore in the source ReplicationController", func() {
				// Deploy a ReplicationController
				rc := deployRC(fmt.Sprintf("source-rc-%s", f.App()))

				// Generate Sample Data
				sampleData := generateSampleData(rc)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Take an Instant Backup the Sample Data
				takeInstantBackup(rc, repo)

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source ReplicationController")
				err = f.CleanupSampleDataFromWorkload(rc.ObjectMeta, apis.KindReplicationController)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original ReplicationController")
				restoredData := restoreData(rc, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Restore in different ReplicationController", func() {

			It("should restore backed up data into different ReplicationController", func() {
				// Deploy a ReplicationController
				rc := deployRC(fmt.Sprintf("source-rc-%s", f.App()))

				// Generate Sample Data
				sampleData := generateSampleData(rc)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Take an Instant Backup the Sample Data
				takeInstantBackup(rc, repo)

				// Deploy restored ReplicationController
				restoredRC := deployRC(fmt.Sprintf("restored-rc-%s", f.App()))

				// Restore the backup data
				By("Restoring the backed up data in the restored ReplicationController")
				restoredData := restoreData(restoredRC, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Leader election for backup and restore ReplicationController", func() {

			It("Should leader elect and backup and restore ReplicationController", func() {
				// Deploy a ReplicationController
				rc := deployRCWithMultipleReplica(fmt.Sprintf("source-rc-%s", f.App()))

				//  Generate Sample Data
				sampleData := generateSampleData(rc)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := setupRCBackup(rc, repo)

				By("Waiting for leader election")
				f.CheckLeaderElection(rc.ObjectMeta, apis.KindReplicationController, v1beta1.ResourceKindBackupConfiguration)

				// Create Instant BackupSession for Trigger Instant Backup
				By("Triggering Instant Backup")
				backupSession, err := f.TriggerInstantBackup(backupConfig)
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(backupSession)

				By("Waiting for backup process to complete")
				f.EventuallyBackupProcessCompleted(backupSession.ObjectMeta).Should(BeTrue())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source ReplicationController")
				err = f.CleanupSampleDataFromWorkload(rc.ObjectMeta, apis.KindReplicationController)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original ReplicationController")
				restoredData := restoreData(rc, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

	})

})
