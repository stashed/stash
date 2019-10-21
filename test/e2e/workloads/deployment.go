package workloads

import (
	"fmt"
	"strings"

	"github.com/appscode/go/sets"
	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apps_util "kmodules.xyz/client-go/apps/v1"
	"stash.appscode.dev/stash/apis"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"
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

		deployDeployment = func(name string) *apps.Deployment {
			// Create PVC for Deployment
			pvc := createPVC(name)
			// Generate Deployment definition
			deployment := f.Deployment(pvc.Name)
			deployment.Name = name

			By("Deploying Deployment: " + deployment.Name)
			createdDeployment, err := f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdDeployment)

			By("Waiting for Deployment to be ready")
			err = apps_util.WaitUntilDeploymentReady(f.KubeClient, createdDeployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdDeployment.ObjectMeta).Should(BeTrue())

			return createdDeployment
		}

		deployDeploymentWithMultipleReplica = func(name string) *apps.Deployment {
			// Create PVC for Deployment
			pvc := createPVC(fmt.Sprintf("source-pvc-%s", f.App()))
			// Generate Deployment definition
			deployment := f.Deployment(pvc.Name)
			deployment.Spec.Replicas = types.Int32P(2) // two replicas
			deployment.Name = name

			By("Deploying Deployment: " + deployment.Name)
			createdDeployment, err := f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdDeployment)

			By("Waiting for Deployment to be ready")
			err = apps_util.WaitUntilDeploymentReady(f.KubeClient, createdDeployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdDeployment.ObjectMeta).Should(BeTrue())

			return createdDeployment
		}

		generateSampleData = func(deployment *apps.Deployment) sets.String {
			By("Generating sample data inside Deployment pods")
			err := f.CreateSampleDataInsideWorkload(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that sample data has been generated")
			sampleData, err := f.ReadSampleDataFromFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleData).ShouldNot(BeEmpty())

			return sampleData
		}

		getTargetRef = func(deployment *apps.Deployment) v1beta1.TargetRef {
			return v1beta1.TargetRef{
				Name:       deployment.Name,
				Kind:       apis.KindDeployment,
				APIVersion: "apps/v1",
			}
		}

		setupDeploymentBackup = func(deployment *apps.Deployment, repo *api.Repository) *v1beta1.BackupConfiguration {
			// Generate desired BackupConfiguration definition
			backupConfig := f.GetBackupConfigurationForWorkload(repo.Name, getTargetRef(deployment))

			By("Creating BackupConfiguration: " + backupConfig.Name)
			createdBC, err := f.StashClient.StashV1beta1().BackupConfigurations(backupConfig.Namespace).Create(backupConfig)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdBC)

			By("Verifying that backup triggering CronJob has been created")
			f.EventuallyCronJobCreated(backupConfig.ObjectMeta).Should(BeTrue())

			By("Verifying that sidecar has been injected")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Deployment to be ready with sidecar")
			err = f.WaitUntilDeploymentReadyWithSidecar(deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			return createdBC
		}

		takeInstantBackup = func(deployment *apps.Deployment, repo *api.Repository) {
			// Setup Backup
			backupConfig := setupDeploymentBackup(deployment, repo)

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

		restoreData = func(deployment *apps.Deployment, repo *api.Repository) sets.String {
			By("Creating RestoreSession")
			restoreSession := f.GetRestoreSessionForWorkload(repo.Name, getTargetRef(deployment))
			err := f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(restoreSession)

			By("Verifying that init-container has been injected")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveInitContainer(util.StashInitContainer))

			By("Waiting for restore process to complete")
			f.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta).Should(BeTrue())

			By("Verifying that RestoreSession succeeded")
			completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

			By("Waiting for Deployment to be ready with init-container")
			err = f.WaitUntilDeploymentReadyWithInitContainer(deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			f.EventuallyPodAccessible(deployment.ObjectMeta).Should(BeTrue())

			By("Reading restored data")
			restoredData, err := f.ReadSampleDataFromFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(restoredData).NotTo(BeEmpty())

			return restoredData
		}
	)

	Context("Deployment", func() {

		Context("Restore in same Deployment", func() {

			It("should Backup & Restore in the source Deployment", func() {
				// Deploy a Deployment
				deployment := deployDeployment(fmt.Sprintf("source-deployment-%s", f.App()))

				// Generate Sample Data
				sampleData := generateSampleData(deployment)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Take an Instant Backup the Sample Data
				takeInstantBackup(deployment, repo)

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source Deployment")
				err = f.CleanupSampleDataFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original Deployment")
				restoredData := restoreData(deployment, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Restore in different Deployment", func() {

			It("should restore backed up data into different Deployment", func() {
				// Deploy a Deployment
				deployment := deployDeployment(fmt.Sprintf("source-deployment-%s", f.App()))

				// Generate Sample Data
				sampleData := generateSampleData(deployment)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Take an Instant Backup the Sample Data
				takeInstantBackup(deployment, repo)

				// Deploy restored Deployment
				restoredDeployment := deployDeployment(fmt.Sprintf("restored-deployment-%s", f.App()))

				// Restore the backup data
				By("Restoring the backed up data in the restored Deployment")
				restoredData := restoreData(restoredDeployment, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Leader election for backup and restore Deployment", func() {

			It("Should leader elect and backup and restore Deployment", func() {
				// Deploy a Deployment
				deployment := deployDeploymentWithMultipleReplica(fmt.Sprintf("source-deployment-%s", f.App()))

				//  Generate Sample Data
				sampleData := generateSampleData(deployment)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := setupDeploymentBackup(deployment, repo)

				By("Waiting for leader election")
				f.CheckLeaderElection(deployment.ObjectMeta, apis.KindDeployment, v1beta1.ResourceKindBackupConfiguration)

				// Create Instant BackupSession Trigger Instant Backup
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
				By("Deleting sample data from source Deployment")
				err = f.CleanupSampleDataFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original Deployment")
				restoredData := restoreData(deployment, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

	})

})
