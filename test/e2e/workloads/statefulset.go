/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package workloads

import (
	"context"
	"fmt"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Workload Test", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	JustAfterEach(func() {
		f.PrintDebugInfoOnFailure()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
		// StatefulSet's PVCs are not get cleanup by the CleanupTestResources() function.
		// Hence, we need to cleanup them manually.
		f.CleanupUndeletedPVCs()
	})

	Context("StatefulSet", func() {

		Context("Restore in same StatefulSet", func() {
			It("should Backup & Restore in the source StatefulSet", func() {
				// Deploy a StatefulSet
				ss, err := f.DeployStatefulSet(framework.SourceStatefulSet, int32(3), framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(ss.ObjectMeta, repo, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup of the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta, v1beta1.BackupInvokerRef{
					Name: backupConfig.Name,
					Kind: v1beta1.ResourceKindBackupConfiguration,
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(context.TODO(), backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source StatefulSet")
				err = f.CleanupSampleDataFromWorkload(ss.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data in the original StatefulSet")
				restoreSession, err := f.SetupRestoreProcess(ss.ObjectMeta, repo, apis.KindStatefulSet, framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

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
				ss, err := f.DeployStatefulSet(framework.SourceStatefulSet, int32(3), framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(ss.ObjectMeta, repo, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup of the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta, v1beta1.BackupInvokerRef{
					Name: backupConfig.Name,
					Kind: v1beta1.ResourceKindBackupConfiguration,
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(context.TODO(), backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Deploy restored StatefulSet
				restoredSS, err := f.DeployStatefulSet(framework.RestoredStatefulSet, int32(3), framework.RestoredVolume)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data in the new StatefulSet")
				restoreSession, err := f.SetupRestoreProcess(restoredSS.ObjectMeta, repo, apis.KindStatefulSet, framework.RestoredVolume)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

				// Get restored data
				restoredData := f.RestoredData(restoredSS.ObjectMeta, apis.KindStatefulSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Restore on scaled up StatefulSet", func() {
			It("should restore the backed up data into scaled up StatefulSet", func() {
				// Deploy a StatefulSet
				ss, err := f.DeployStatefulSet(framework.SourceStatefulSet, int32(3), framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(ss.ObjectMeta, repo, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup of the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta, v1beta1.BackupInvokerRef{
					Name: backupConfig.Name,
					Kind: v1beta1.ResourceKindBackupConfiguration,
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(context.TODO(), backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Deploy restored StatefulSet
				restoredSS, err := f.DeployStatefulSet(framework.RestoredStatefulSet, int32(5), framework.RestoredVolume)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data in different StatefulSet")
				restoreSession, err := f.SetupRestoreProcess(restoredSS.ObjectMeta, repo, apis.KindStatefulSet, framework.RestoredVolume, func(restore *v1beta1.RestoreSession) {
					restore.Spec.Target.Rules = []v1beta1.Rule{
						{
							TargetHosts: []string{
								fmt.Sprintf("%s-3", f.App()),
								fmt.Sprintf("%s-4", f.App()),
							},
							SourceHost: fmt.Sprintf("%s-0", f.App()),
							Paths: []string{
								framework.TestSourceDataMountPath,
							},
						},
						{
							TargetHosts: []string{},
							Paths: []string{
								framework.TestSourceDataMountPath,
							},
						},
					}
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

				// Get restored data
				restoredData := f.RestoredData(restoredSS.ObjectMeta, apis.KindStatefulSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})
	})
})
