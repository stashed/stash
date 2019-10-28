package workloads

import (
	"fmt"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	apps_util "kmodules.xyz/client-go/apps/v1"
)

var _ = Describe("Workload Test", func() {

	var f *framework.Invocation

	var (
		deployDaemonSet = func(name string) *apps.DaemonSet {
			// Generate DaemonSet definition
			dmn := f.DaemonSet()
			dmn.Name = name

			By("Deploying DaemonSet: " + dmn.Name)
			createdDmn, err := f.CreateDaemonSet(dmn)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdDmn)

			By("Waiting for DaemonSet to be ready")
			err = apps_util.WaitUntilDaemonSetReady(f.KubeClient, dmn.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(dmn.ObjectMeta).Should(BeTrue())

			return createdDmn
		}
	)

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("DaemonSet", func() {

		Context("Restore in same DaemonSet", func() {
			It("should Backup & Restore in the source DaemonSet", func() {
				// Deploy a DaemonSet
				dmn := deployDaemonSet(fmt.Sprintf("source-daemon-%s", f.App()))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(dmn.ObjectMeta, apis.KindDaemonSet)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := f.SetupWorkloadBackup(dmn.ObjectMeta, repo, apis.KindDaemonSet)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source DaemonSet")
				err = f.CleanupSampleDataFromWorkload(dmn.ObjectMeta, apis.KindDaemonSet)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original DaemonSet")
				restoredData := f.RestoreData(dmn.ObjectMeta, repo, apis.KindDaemonSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Restore in different DaemonSet", func() {
			It("should restore backed up data into different DaemonSet", func() {
				// Deploy a DaemonSet
				dmn := deployDaemonSet(fmt.Sprintf("source-daemon-%s", f.App()))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(dmn.ObjectMeta, apis.KindDaemonSet)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := f.SetupWorkloadBackup(dmn.ObjectMeta, repo, apis.KindDaemonSet)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)

				// Deploy restored DaemonSet
				restoredDmn := deployDaemonSet(fmt.Sprintf("restored-daemon-%s", f.App()))

				// Restore the backup data
				By("Restoring the backed up data in the restored DaemonSet")
				restoredData := f.RestoreData(restoredDmn.ObjectMeta, repo, apis.KindDaemonSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})
	})
})
