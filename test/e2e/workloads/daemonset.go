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

	"github.com/appscode/go/sets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

		generateSampleData = func(dmn *apps.DaemonSet) sets.String {
			By("Generating sample data inside daemon pods")
			err := f.CreateSampleDataInsideWorkload(dmn.ObjectMeta, apis.KindDaemonSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that sample data has been generated")
			sampleData, err := f.ReadSampleDataFromFromWorkload(dmn.ObjectMeta, apis.KindDaemonSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleData).ShouldNot(BeEmpty())

			return sampleData
		}

		getTargetRef = func(dmn *apps.DaemonSet) v1beta1.TargetRef {
			return v1beta1.TargetRef{
				Name:       dmn.Name,
				Kind:       apis.KindDaemonSet,
				APIVersion: "apps/v1",
			}
		}

		setupDaemonBackup = func(dmn *apps.DaemonSet, repo *api.Repository) *v1beta1.BackupConfiguration {
			// Generate desired BackupConfiguration definition
			backupConfig := f.GetBackupConfigurationForWorkload(repo.Name, getTargetRef(dmn))

			By("Creating BackupConfiguration: " + backupConfig.Name)
			createdBC, err := f.StashClient.StashV1beta1().BackupConfigurations(backupConfig.Namespace).Create(backupConfig)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdBC)

			By("Verifying that backup triggering CronJob has been created")
			f.EventuallyCronJobCreated(backupConfig.ObjectMeta).Should(BeTrue())

			By("Verifying that sidecar has been injected")
			f.EventuallyDaemonSet(dmn.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for DaemonSet to be ready with sidecar")
			err = f.WaitUntilDaemonSetReadyWithSidecar(dmn.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			return createdBC
		}

		takeInstantBackup = func(dmn *apps.DaemonSet, repo *api.Repository) {
			// Setup Backup
			backupConfig := setupDaemonBackup(dmn, repo)

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

		restoreData = func(dmn *apps.DaemonSet, repo *api.Repository) sets.String {
			By("Creating RestoreSession")
			restoreSession := f.GetRestoreSessionForWorkload(repo.Name, getTargetRef(dmn))
			err := f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(restoreSession)

			By("Verifying that init-container has been injected")
			f.EventuallyDaemonSet(dmn.ObjectMeta).Should(HaveInitContainer(util.StashInitContainer))

			By("Waiting for restore process to complete")
			f.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta).Should(BeTrue())

			By("Verifying that RestoreSession succeeded")
			completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

			By("Waiting for DaemonSet to be ready with init-container")
			err = f.WaitUntilDaemonSetReadyWithInitContainer(dmn.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			f.EventuallyPodAccessible(dmn.ObjectMeta).Should(BeTrue())

			By("Reading restored data")
			restoredData, err := f.ReadSampleDataFromFromWorkload(dmn.ObjectMeta, apis.KindDaemonSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(restoredData).NotTo(BeEmpty())

			return restoredData
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
				sampleData := generateSampleData(dmn)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Take an Instant Backup the Sample Data
				takeInstantBackup(dmn, repo)

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source DaemonSet")
				err = f.CleanupSampleDataFromWorkload(dmn.ObjectMeta, apis.KindDaemonSet)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original DaemonSet")
				restoredData := restoreData(dmn, repo)

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
				sampleData := generateSampleData(dmn)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Take an Instant Backup the Sample Data
				takeInstantBackup(dmn, repo)

				// Deploy restored DaemonSet
				restoredDmn := deployDaemonSet(fmt.Sprintf("restored-daemon-%s", f.App()))

				// Restore the backup data
				By("Restoring the backed up data in the restored DaemonSet")
				restoredData := restoreData(restoredDmn, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})
	})
})
