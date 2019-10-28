package workloads

import (
	"fmt"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
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
		deployRC = func(name string) *core.ReplicationController {
			// Create PVC for ReplicationController
			pvc := f.CreateNewPVC(name)
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
			pvc := f.CreateNewPVC(name)
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
	)

	Context("ReplicationController", func() {

		Context("Restore in same ReplicationController", func() {
			It("should Backup & Restore in the source ReplicationController", func() {
				// Deploy a ReplicationController
				rc := deployRC(fmt.Sprintf("source-rc-%s", f.App()))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(rc.ObjectMeta, apis.KindReplicationController)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := f.SetupWorkloadBackup(rc.ObjectMeta, repo, apis.KindReplicationController)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source ReplicationController")
				err = f.CleanupSampleDataFromWorkload(rc.ObjectMeta, apis.KindReplicationController)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original ReplicationController")
				restoredData := f.RestoreData(rc.ObjectMeta, repo, apis.KindReplicationController)

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
				sampleData := f.GenerateSampleData(rc.ObjectMeta, apis.KindReplicationController)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := f.SetupWorkloadBackup(rc.ObjectMeta, repo, apis.KindReplicationController)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)

				// Deploy restored ReplicationController
				restoredRC := deployRC(fmt.Sprintf("restored-rc-%s", f.App()))

				// Restore the backup data
				By("Restoring the backed up data in the restored ReplicationController")
				restoredData := f.RestoreData(restoredRC.ObjectMeta, repo, apis.KindReplicationController)

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
				sampleData := f.GenerateSampleData(rc.ObjectMeta, apis.KindReplicationController)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := f.SetupWorkloadBackup(rc.ObjectMeta, repo, apis.KindReplicationController)

				By("Waiting for leader election")
				f.CheckLeaderElection(rc.ObjectMeta, apis.KindReplicationController, v1beta1.ResourceKindBackupConfiguration)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source ReplicationController")
				err = f.CleanupSampleDataFromWorkload(rc.ObjectMeta, apis.KindReplicationController)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original ReplicationController")
				restoredData := f.RestoreData(rc.ObjectMeta, repo, apis.KindReplicationController)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

	})

})
