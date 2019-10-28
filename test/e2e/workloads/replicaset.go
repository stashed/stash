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

var _ = Describe("ReplicaSet", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	var (
		deployRS = func(name string) *apps.ReplicaSet {
			// Create PVC for ReplicaSet
			pvc := f.CreateNewPVC(name)
			// Generate ReplicaSet definition
			rs := f.ReplicaSet(pvc.Name)
			rs.Name = name

			By("Deploying ReplicaSet: " + rs.Name)
			createdRS, err := f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdRS)

			By("Waiting for ReplicaSet to be ready")
			err = apps_util.WaitUntilReplicaSetReady(f.KubeClient, createdRS.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdRS.ObjectMeta).Should(BeTrue())

			return createdRS
		}

		deployRSWithMultipleReplica = func(name string) *apps.ReplicaSet {
			// Create PVC for ReplicaSet
			pvc := f.CreateNewPVC(fmt.Sprintf(name))
			// Generate ReplicaSet definition
			rs := f.ReplicaSet(pvc.Name)
			rs.Spec.Replicas = types.Int32P(2) // two replicas
			rs.Name = name

			By("Deploying ReplicaSet: " + rs.Name)
			createdRS, err := f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdRS)

			By("Waiting for ReplicaSet to be ready")
			err = apps_util.WaitUntilReplicaSetReady(f.KubeClient, createdRS.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdRS.ObjectMeta).Should(BeTrue())

			return createdRS
		}
	)

	Context("ReplicaSet", func() {

		Context("Restore in same ReplicaSet", func() {
			It("should Backup & Restore in the source ReplicaSet", func() {
				// Deploy a ReplicaSet
				rs := deployRS(fmt.Sprintf("source-rs-%s", f.App()))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := f.SetupWorkloadBackup(rs.ObjectMeta, repo, apis.KindReplicaSet)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source ReplicaSet")
				err = f.CleanupSampleDataFromWorkload(rs.ObjectMeta, apis.KindReplicaSet)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original ReplicaSet")
				restoredData := f.RestoreData(rs.ObjectMeta, repo, apis.KindReplicaSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Restore in different ReplicaSet", func() {
			It("should restore backed up data into different ReplicaSet", func() {
				// Deploy a ReplicaSet
				rs := deployRS(fmt.Sprintf("source-rs-%s", f.App()))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := f.SetupWorkloadBackup(rs.ObjectMeta, repo, apis.KindReplicaSet)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)

				// Deploy restored ReplicaSet
				restoredRS := deployRS(fmt.Sprintf("restored-rs-%s", f.App()))

				// Restore the backup data
				By("Restoring the backed up data in the restored ReplicaSet")
				restoredData := f.RestoreData(restoredRS.ObjectMeta, repo, apis.KindReplicaSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Leader election for backup and restore ReplicaSet", func() {
			It("Should leader elect and backup and restore ReplicaSet", func() {
				// Deploy a ReplicaSet
				rs := deployRSWithMultipleReplica(fmt.Sprintf("source-rs-multiple-%s", f.App()))

				//  Generate Sample Data
				sampleData := f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := f.SetupWorkloadBackup(rs.ObjectMeta, repo, apis.KindReplicaSet)

				By("Waiting for leader election")
				f.CheckLeaderElection(rs.ObjectMeta, apis.KindReplicaSet, v1beta1.ResourceKindBackupConfiguration)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source ReplicaSet")
				err = f.CleanupSampleDataFromWorkload(rs.ObjectMeta, apis.KindReplicaSet)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original ReplicaSet")
				restoredData := f.RestoreData(rs.ObjectMeta, repo, apis.KindReplicaSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

	})

})
