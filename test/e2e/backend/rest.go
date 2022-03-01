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

package backend

import (
	"context"
	"net"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Rest Backend", func() {
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

	Context("Without TLS", func() {
		BeforeEach(func() {
			By("Creating REST Server")
			_, err := f.CreateRestServer(false, []net.IP{net.ParseIP("127.0.0.1")})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should backup and restore sample data", func() {
			// Deploy a Deployment
			deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())

			// Generate Sample Data
			sampleData, err := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Setup a Rest Repository
			repo, err := f.SetupRestRepository(false, framework.TEST_REST_SERVER_USERNAME, framework.TEST_REST_SERVER_PASSWORD)
			Expect(err).NotTo(HaveOccurred())
			//
			// Setup workload Backup
			backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment)
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
			//
			// Simulate disaster scenario. Delete the data from source PVC
			By("Deleting sample data from source Deployment")
			err = f.CleanupSampleDataFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Restore the backed up data
			By("Restoring the backed up data in the original Deployment")
			restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment, framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that RestoreSession succeeded")
			completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

			// Get restored data
			restoredData := f.RestoredData(deployment.ObjectMeta, apis.KindDeployment)

			// Verify that restored data is same as the original data
			By("Verifying restored data is same as the original data")
			Expect(restoredData).Should(BeSameAs(sampleData))
		})
	})

	Context("With TLS", func() {
		BeforeEach(func() {
			By("Creating REST Server")
			_, err := f.CreateRestServer(true, []net.IP{net.ParseIP("127.0.0.1")})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should backup and restore sample data", func() {
			// Deploy a Deployment
			deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())

			// Generate Sample Data
			sampleData, err := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Setup a Rest Repository
			repo, err := f.SetupRestRepository(true, framework.TEST_REST_SERVER_USERNAME, framework.TEST_REST_SERVER_PASSWORD)
			Expect(err).NotTo(HaveOccurred())
			//
			// Setup workload Backup
			backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment)
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
			//
			// Simulate disaster scenario. Delete the data from source PVC
			By("Deleting sample data from source Deployment")
			err = f.CleanupSampleDataFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Restore the backed up data
			By("Restoring the backed up data in the original Deployment")
			restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment, framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that RestoreSession succeeded")
			completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

			// Get restored data
			restoredData := f.RestoredData(deployment.ObjectMeta, apis.KindDeployment)

			// Verify that restored data is same as the original data
			By("Verifying restored data is same as the original data")
			Expect(restoredData).Should(BeSameAs(sampleData))
		})

		It("should backup and restore big file", func() {
			// Deploy a Deployment
			deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())

			// Generate Sample Data
			sampleData, err := f.GenerateBigSampleFile(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Setup a Rest Repository
			repo, err := f.SetupRestRepository(true, framework.TEST_REST_SERVER_USERNAME, framework.TEST_REST_SERVER_PASSWORD)
			Expect(err).NotTo(HaveOccurred())

			// Setup workload Backup
			backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment)
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
			By("Deleting sample data from source Deployment")
			err = f.CleanupSampleDataFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Restore the backed up data
			By("Restoring the backed up data in the original Deployment")
			restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment, framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that RestoreSession succeeded")
			completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

			// Get restored data
			restoredData := f.RestoredData(deployment.ObjectMeta, apis.KindDeployment)

			// Verify that restored data is same as the original data
			By("Verifying restored data is same as the original data")
			Expect(restoredData).Should(BeSameAs(sampleData))
		})
	})
})
