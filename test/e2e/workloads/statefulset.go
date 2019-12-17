/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package workloads

import (
	"fmt"

	"stash.appscode.dev/stash/apis"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("StatefulSet", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			f.PrintDebugHelpers()
		}
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	var (
		setupRestoreProcessOnScaledUpSS = func(ss *apps.StatefulSet, repo *api.Repository) (*v1beta1.RestoreSession, error) {
			By("Creating RestoreSession")
			restoreSession := f.GetRestoreSession(repo.Name, func(restore *v1beta1.RestoreSession) {
				restore.Spec.Target = &v1beta1.RestoreTarget{
					Ref: framework.GetTargetRef(ss.Name, apis.KindStatefulSet),
				}
				restore.Spec.Rules = []v1beta1.Rule{
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
			})

			err := f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(restoreSession)

			By("Verifying that init-container has been injected")
			f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveInitContainer(util.StashInitContainer))

			By("Waiting for StatefulSet to be ready with init-container")
			err = f.WaitUntilStatefulSetWithInitContainer(ss.ObjectMeta)
			f.EventuallyPodAccessible(ss.ObjectMeta).Should(BeTrue())

			By("Waiting for restore process to complete")
			f.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta).Should(BeTrue())

			return restoreSession, err
		}
	)

	Context("StatefulSet", func() {

		Context("Restore in same StatefulSet", func() {
			It("should Backup & Restore in the source StatefulSet", func() {
				// Deploy a StatefulSet
				ss, err := f.DeployStatefulSet(fmt.Sprintf("source-ss1-%s", f.App()), int32(3))
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(ss.ObjectMeta, repo, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup of the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source StatefulSet")
				err = f.CleanupSampleDataFromWorkload(ss.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data in the original StatefulSet")
				restoreSession, err := f.SetupRestoreProcess(ss.ObjectMeta, repo, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

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
				ss, err := f.DeployStatefulSet(fmt.Sprintf("source-ss2-%s", f.App()), int32(3))
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(ss.ObjectMeta, repo, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup of the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Deploy restored StatefulSet
				restoredSS, err := f.DeployStatefulSet(fmt.Sprintf("restored-ss2-%s", f.App()), int32(3))
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data in the original StatefulSet")
				restoreSession, err := f.SetupRestoreProcess(restoredSS.ObjectMeta, repo, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

				// Get restored data
				restoredData := f.RestoredData(restoredSS.ObjectMeta, apis.KindStatefulSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Restore on scaled up StatefulSet", func() {
			It("should restore backed up data into scaled up StatefulSet", func() {
				// Deploy a StatefulSet
				ss, err := f.DeployStatefulSet(fmt.Sprintf("source-ss3-%s", f.App()), int32(3))
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(ss.ObjectMeta, repo, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup of the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Deploy restored StatefulSet
				restoredSS, err := f.DeployStatefulSet(fmt.Sprintf("restored-ss3-%s", f.App()), int32(5))
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data in different StatefulSet")
				restoreSession, err := setupRestoreProcessOnScaledUpSS(restoredSS, repo)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

				// Get restored data
				restoredData := f.RestoredData(restoredSS.ObjectMeta, apis.KindStatefulSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

	})

})
