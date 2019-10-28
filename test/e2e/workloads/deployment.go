package workloads

import (
	"fmt"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	apps_util "kmodules.xyz/client-go/apps/v1"
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
		deployDeployment = func(name string) *apps.Deployment {
			// Create PVC for Deployment
			pvc := f.CreateNewPVC(name)
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
			pvc := f.CreateNewPVC(name)
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
	)

	Context("Deployment", func() {

		Context("Restore in same Deployment", func() {
			It("should Backup & Restore in the source Deployment", func() {
				// Deploy a Deployment
				deployment := deployDeployment(fmt.Sprintf("source-deployment-%s", f.App()))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)
				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source Deployment")
				err = f.CleanupSampleDataFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original Deployment")
				restoredData := f.RestoreData(deployment.ObjectMeta, repo, apis.KindDeployment)

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
				sampleData := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)

				// Deploy restored Deployment
				restoredDeployment := deployDeployment(fmt.Sprintf("restored-deployment-%s", f.App()))

				// Restore the backup data
				By("Restoring the backed up data in the restored Deployment")
				restoredData := f.RestoreData(restoredDeployment.ObjectMeta, repo, apis.KindDeployment)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Leader election for backup and restore Deployment", func() {
			It("Should leader elect and backup and restore Deployment", func() {
				// Deploy a Deployment
				deployment := deployDeploymentWithMultipleReplica(fmt.Sprintf("source-deployment-muplitiple-%s", f.App()))

				//  Generate Sample Data
				sampleData := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment)

				By("Waiting for leader election")
				f.CheckLeaderElection(deployment.ObjectMeta, apis.KindDeployment, v1beta1.ResourceKindBackupConfiguration)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source Deployment")
				err = f.CleanupSampleDataFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original Deployment")
				restoredData := f.RestoreData(deployment.ObjectMeta, repo, apis.KindDeployment)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

	})

})
