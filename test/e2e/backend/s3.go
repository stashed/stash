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

package backend

import (
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("S3 Backend", func() {

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
	})

	Context("General Backup/Restore", func() {
		It("should backup/restore in/from S3 backend", func() {
			// Deploy a Deployment
			deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())

			// Generate Sample Data
			sampleData, err := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Setup a S3 Repository
			repo, err := f.SetupS3Repository(true)
			Expect(err).NotTo(HaveOccurred())

			// Setup workload Backup
			backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Take an Instant Backup of the Sample Data
			backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta, v1beta1.TargetRef{
				Name: backupConfig.Name,
				Kind: v1beta1.ResourceKindBackupConfiguration,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that BackupSession has succeeded")
			completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

			// Simulate disaster scenario. Delete the data from source PVC
			By("Deleting sample data from source Deployment")
			err = f.CleanupSampleDataFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Restore the backed up data
			By("Restoring the backed up data in the original Deployment")
			restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment, framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that RestoreSession succeeded")
			completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

			// Get restored data
			restoredData := f.RestoredData(deployment.ObjectMeta, apis.KindDeployment)

			// Verify that restored data is same as the original data
			By("Verifying restored data is same as the original data")
			Expect(restoredData).Should(BeSameAs(sampleData))
		})
	})

	Context("WipeOut Repository", func() {
		It("should cleanup data from remote repository", func() {
			// Deploy a Deployment
			deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())

			// Generate Sample Data
			_, err = f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Setup a S3 Repository
			repo, err := f.SetupS3Repository(false)
			Expect(err).NotTo(HaveOccurred())

			// Setup workload Backup
			backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Take an Instant Backup of the Sample Data
			backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta, v1beta1.TargetRef{
				Name: backupConfig.Name,
				Kind: v1beta1.ResourceKindBackupConfiguration,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that BackupSession has succeeded")
			completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

			By("Deleting Repository Object")
			err = f.DeleteRepository(repo)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for repository to delete")
			err = framework.WaitUntilRepositoryDeleted(f.StashClient, repo)
			Expect(err).NotTo(HaveOccurred())

			By("Checking remote repository has been deleted")
			items, err := f.BrowseBackendRepository(repo)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(items).Should(BeEmpty())

		})
	})

	Context("Backup/Restore big file", func() {
		It("should backup/restore big file", func() {
			// Deploy a Deployment
			deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())

			// Generate Sample Data
			sampleData, err := f.GenerateBigSampleFile(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Setup a S3 Repository
			repo, err := f.SetupS3Repository(true)
			Expect(err).NotTo(HaveOccurred())

			// Setup workload Backup
			backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Take an Instant Backup of the Sample Data
			backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta, v1beta1.TargetRef{
				Name: backupConfig.Name,
				Kind: v1beta1.ResourceKindBackupConfiguration,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that BackupSession has succeeded")
			completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

			// Simulate disaster scenario. Delete the data from source PVC
			By("Deleting sample data from source Deployment")
			err = f.CleanupSampleDataFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Restore the backed up data
			By("Restoring the backed up data in the original Deployment")
			restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment, framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that RestoreSession succeeded")
			completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

			// Get restored data
			restoredData := f.RestoredData(deployment.ObjectMeta, apis.KindDeployment)

			// Verify that restored data is same as the original data
			By("Verifying restored data is same as the original data")
			Expect(restoredData).Should(BeSameAs(sampleData))
		})
	})

})
