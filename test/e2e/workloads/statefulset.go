package workloads

import (
	"fmt"

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
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apps_util "kmodules.xyz/client-go/apps/v1"
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
		deploySS = func(name string) *apps.StatefulSet {
			// Generate StatefulSet definition
			ss := f.StatefulSetForV1beta1API()
			ss.Name = name

			By("Deploying StatefulSet: " + ss.Name)
			createdss, err := f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdss)

			By("Waiting for StatefulSet to be ready")
			err = apps_util.WaitUntilStatefulSetReady(f.KubeClient, createdss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdss.ObjectMeta).Should(BeTrue())

			return createdss
		}

		deployScaledUpSS = func(name string) *apps.StatefulSet {
			// Generate StatefulSet definition
			ss := f.StatefulSetForV1beta1API()
			ss.Name = name
			// scaled up StatefulSet
			ss.Spec.Replicas = types.Int32P(5)

			By("Deploying StatefulSet: " + ss.Name)
			createdss, err := f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdss)

			By("Waiting for StatefulSet to be ready")
			err = apps_util.WaitUntilStatefulSetReady(f.KubeClient, createdss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdss.ObjectMeta).Should(BeTrue())

			return createdss
		}

		restoreDataOnScaledUpSS = func(ss *apps.StatefulSet, repo *api.Repository) sets.String {
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

			By("Waiting for restore process to complete")
			f.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta).Should(BeTrue())

			By("Verifying that RestoreSession succeeded")
			completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

			By("Waiting for StatefulSet to be ready with init-container")
			err = f.WaitUntilStatefulSetWithInitContainer(ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			f.EventuallyPodAccessible(ss.ObjectMeta).Should(BeTrue())

			By("Reading restored data")
			restoredData, err := f.ReadSampleDataFromFromWorkload(ss.ObjectMeta, apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(restoredData).NotTo(BeEmpty())

			return restoredData
		}
	)

	Context("StatefulSet", func() {

		Context("Restore in same StatefulSet", func() {
			It("should Backup & Restore in the source StatefulSet", func() {
				// Deploy a StatefulSet
				ss := deploySS(fmt.Sprintf("source-ss-%s", f.App()))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := f.SetupWorkloadBackup(ss.ObjectMeta, repo, apis.KindStatefulSet)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source StatefulSet")
				err = f.CleanupSampleDataFromWorkload(ss.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original StatefulSet")
				restoredData := f.RestoreData(ss.ObjectMeta, repo, apis.KindStatefulSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(Equal(sampleData))
			})
		})

		Context("Restore in different StatefulSet", func() {
			It("should restore backed up data into different StatefulSet", func() {
				// Deploy a StatefulSet
				ss := deploySS(fmt.Sprintf("source-ss-%s", f.App()))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := f.SetupWorkloadBackup(ss.ObjectMeta, repo, apis.KindStatefulSet)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)

				// Deploy restored StatefulSet
				restoredSS := deploySS(fmt.Sprintf("restored-ss-%s", f.App()))

				// Restore the backup data
				By("Restoring the backed up data in the restored StatefulSet")
				restoredData := f.RestoreData(restoredSS.ObjectMeta, repo, apis.KindStatefulSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(Equal(sampleData))
			})
		})

		Context("Restore on scaled up StatefulSet", func() {
			It("should restore backed up data into scaled up StatefulSet", func() {
				// Deploy a StatefulSet
				ss := deploySS(fmt.Sprintf("source-ss-%s", f.App()))

				// Generate Sample Data
				sampleData := f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := f.SetupWorkloadBackup(ss.ObjectMeta, repo, apis.KindStatefulSet)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)

				// Deploy restored StatefulSet
				restoredSS := deployScaledUpSS(fmt.Sprintf("restored-ss-%s", f.App()))

				// Restore the backup data
				By("Restoring the backed up data in the original StatefulSet")
				restoredData := restoreDataOnScaledUpSS(restoredSS, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

	})

})
