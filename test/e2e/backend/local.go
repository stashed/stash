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
	"fmt"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kmodules.xyz/offshoot-api/api/v1"
)

var _ = Describe("Local Backend", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
		By("Creating NFS server")
		_, err := f.CreateNFSServer()
		Expect(err).NotTo(HaveOccurred())

	})

	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			f.PrintDebugHelpers()
			framework.TestFailed = true
		}
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
		By("Deleting NFS server")
		err = f.DeleteNFSServer()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("PVC as backend", func() {
		Context("General Backup/Restore", func() {
			It("should backup/restore in/from Local backend", func() {
				// Deploy a Deployment
				deployment, err := f.DeployDeployment(fmt.Sprintf("source-deployment-%s", f.App()), int32(1))
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Local Repository
				repo, err := f.SetupLocalRepositoryWithPVC()
				Expect(err).NotTo(HaveOccurred())

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
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
				restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment)
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

		Context("Backup/Restore big file", func() {
			It("should backup/restore big file", func() {
				// Deploy a Deployment
				deployment, err := f.DeployDeployment(fmt.Sprintf("source-deployment-%s", f.App()), int32(1))
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateBigSampleFile(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Local Repository
				repo, err := f.SetupLocalRepositoryWithPVC()
				Expect(err).NotTo(HaveOccurred())

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
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
				restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment)
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

	Context("NFS", func() {
		Context("General Backup/Restore", func() {
			It("should backup/restore in/from Local backend", func() {
				// Deploy a Deployment
				deployment, err := f.DeployDeployment(fmt.Sprintf("source-deployment-%s", f.App()), int32(1), func(dp *apps.Deployment) {
					dp.Spec.Template.Spec.Containers[0].SecurityContext = &core.SecurityContext{
						Privileged: types.BoolP(true),
						RunAsUser:  types.Int64P(int64(0)),
						RunAsGroup: types.Int64P(int64(0)),
					}
				})
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Local Repository
				repo, err := f.SetupLocalRepositoryWithNFSServer()
				Expect(err).NotTo(HaveOccurred())

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
					bc.Spec.RuntimeSettings.Container = &v1.ContainerRuntimeSettings{
						SecurityContext: &core.SecurityContext{
							Privileged: types.BoolP(true),
							RunAsUser:  types.Int64P(int64(0)),
							RunAsGroup: types.Int64P(int64(0)),
						},
					}
				})
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
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
				restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment)
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

		Context("Backup/Restore big file", func() {
			It("should backup/restore big file", func() {
				// Deploy a Deployment
				deployment, err := f.DeployDeployment(fmt.Sprintf("source-deployment-%s", f.App()), int32(1), func(dp *apps.Deployment) {
					dp.Spec.Template.Spec.Containers[0].SecurityContext = &core.SecurityContext{
						Privileged: types.BoolP(true),
						RunAsUser:  types.Int64P(int64(0)),
						RunAsGroup: types.Int64P(int64(0)),
					}
				})
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateBigSampleFile(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Local Repository
				repo, err := f.SetupLocalRepositoryWithNFSServer()
				Expect(err).NotTo(HaveOccurred())

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
					bc.Spec.RuntimeSettings.Container = &v1.ContainerRuntimeSettings{
						SecurityContext: &core.SecurityContext{
							Privileged: types.BoolP(true),
							RunAsUser:  types.Int64P(int64(0)),
							RunAsGroup: types.Int64P(int64(0)),
						},
					}
				})
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
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
				restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment)
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

	Context("HostPath", func() {
		Context("General Backup/Restore", func() {
			It("should backup/restore in/from Local backend", func() {
				// Deploy a Deployment
				deployment, err := f.DeployDeployment(fmt.Sprintf("source-deployment-%s", f.App()), int32(1))
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Local Repository
				repo, err := f.SetupLocalRepositoryWithHostPath()
				Expect(err).NotTo(HaveOccurred())

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
					bc.Spec.RuntimeSettings.Container = &v1.ContainerRuntimeSettings{
						SecurityContext: &core.SecurityContext{
							Privileged: types.BoolP(true),
							RunAsUser:  types.Int64P(int64(0)),
							RunAsGroup: types.Int64P(int64(0)),
						},
					}
				})
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
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
				restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment)
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

		Context("Backup/Restore big file", func() {
			It("should backup/restore big file", func() {
				// Deploy a Deployment
				deployment, err := f.DeployDeployment(fmt.Sprintf("source-deployment-%s", f.App()), int32(1))
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateBigSampleFile(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Local Repository
				repo, err := f.SetupLocalRepositoryWithPVC()
				Expect(err).NotTo(HaveOccurred())

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
					bc.Spec.RuntimeSettings.Container = &v1.ContainerRuntimeSettings{
						SecurityContext: &core.SecurityContext{
							Privileged: types.BoolP(true),
							RunAsUser:  types.Int64P(int64(0)),
							RunAsGroup: types.Int64P(int64(0)),
						},
					}
				})
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
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
				restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment)
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

})
