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

package misc

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pfutil "kmodules.xyz/client-go/tools/portforward"
)

var _ = Describe("Backward Compatibility Test", func() {

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

	Context("Alias Missing", func() {
		Context("Deployment", func() {
			It("should generate hostname automatically", func() {
				// Deploy a Deployment
				deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
					bc.Spec.Target.Alias = ""
				})
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
				restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment, framework.SourceVolume, func(restore *v1beta1.RestoreSession) {
					restore.Spec.Target.Alias = ""
				})
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

		Context("StatefulSet", func() {
			It("should generate hostname automatically", func() {
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
				backupConfig, err := f.SetupWorkloadBackup(ss.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
					bc.Spec.Target.Alias = ""
				})
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
					restore.Spec.Target.Alias = ""
					restore.Spec.Target.Rules = []v1beta1.Rule{
						{
							TargetHosts: []string{"host-3", "host-4"},
							SourceHost:  "host-0",
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

		Context("DaemonSet", func() {
			It("should generate hostname automatically", func() {
				// Deploy a DaemonSet
				dmn, err := f.DeployDaemonSet(framework.SourceDaemonSet, framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(dmn.ObjectMeta, apis.KindDaemonSet)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(dmn.ObjectMeta, repo, apis.KindDaemonSet, func(bc *v1beta1.BackupConfiguration) {
					bc.Spec.Target.Alias = ""
				})
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
				By("Deleting sample data from source DaemonSet")
				err = f.CleanupSampleDataFromWorkload(dmn.ObjectMeta, apis.KindDaemonSet)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data in the original DaemonSet")
				restoreSession, err := f.SetupRestoreProcess(dmn.ObjectMeta, repo, apis.KindDaemonSet, framework.SourceVolume, func(restore *v1beta1.RestoreSession) {
					restore.Spec.Target.Alias = ""
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

				// Get restored data
				restoredData := f.RestoredData(dmn.ObjectMeta, apis.KindDaemonSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("PVC", func() {
			It("should generate hostname automatically", func() {
				// Create new PVC
				pvc, err := f.CreateNewPVC(fmt.Sprintf("%s-%s", framework.SourceVolume, f.App()))
				Expect(err).NotTo(HaveOccurred())

				// Deploy a Pod
				pod, err := f.DeployPod(pvc.Name)
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup PVC Backup
				backupConfig, err := f.SetupPVCBackup(pvc, repo, func(bc *v1beta1.BackupConfiguration) {
					bc.Spec.Target.Alias = ""
				})
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
				By("Deleting sample data from source Pod")
				err = f.CleanupSampleDataFromWorkload(pod.ObjectMeta, apis.KindPod)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data")
				restoreSession, err := f.SetupRestoreProcessForPVC(pvc, repo, func(restore *v1beta1.RestoreSession) {
					restore.Spec.Target.Alias = ""
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

				// Get restored data
				restoredData := f.RestoredData(pod.ObjectMeta, apis.KindPod)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Database", func() {
			It("should generate hostname automatically", func() {
				// Deploy MySQL database and respective service,secret,PVC and AppBinding.
				By("Deploying MySQL Server")
				dpl, appBinding, err := f.DeployMySQLDatabase()
				Expect(err).NotTo(HaveOccurred())

				By("Port forwarding MySQL pod")
				pod, err := f.GetPod(dpl.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				tunnel := pfutil.NewTunnel(f.KubeClient.CoreV1().RESTClient(), f.ClientConfig, pod.Namespace, pod.Name, framework.MySQLServingPortNumber)
				defer tunnel.Close()
				err = tunnel.ForwardPort()
				Expect(err).NotTo(HaveOccurred())

				By("Connecting with MySQL Server")
				connstr := fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql", framework.SuperUser, f.App(), framework.LocalHostIP, tunnel.Local)
				db, err := sql.Open("mysql", connstr)
				Expect(err).NotTo(HaveOccurred())
				defer db.Close()
				db.SetConnMaxLifetime(time.Second * 10)
				err = f.EventuallyConnectWithMySQLServer(db)
				Expect(err).NotTo(HaveOccurred())

				By("Creating Sample Table")
				err = f.CreateTable(db, sampleTable)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the sample table has been created")
				tables, err := f.ListTables(db)
				Expect(err).NotTo(HaveOccurred())
				Expect(tables.Has(sampleTable)).Should(BeTrue())

				By("Insert sample data")
				property := "price"
				originalValue := 123456
				err = f.InsertRow(db, sampleTable, property, originalValue)
				Expect(err).NotTo(HaveOccurred())

				By("Verify that sample data has been inserted")
				res, err := f.ReadProperty(db, sampleTable, property)
				Expect(err).NotTo(HaveOccurred())
				Expect(res).Should(BeEquivalentTo(originalValue))

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Database Backup
				backupConfig, err := f.SetupDatabaseBackup(appBinding, repo, func(bc *v1beta1.BackupConfiguration) {
					bc.Spec.Target.Alias = ""
				})
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

				// Simulate disaster
				// Update data of the sample table
				By("Updating data to simulate disaster")
				err = f.UpdateProperty(db, sampleTable, property, 0)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the data has been updated")
				res, err = f.ReadProperty(db, sampleTable, property)
				Expect(err).NotTo(HaveOccurred())
				Expect(res).ShouldNot(BeEquivalentTo(originalValue))

				// Restore the backed up data
				// Cleanup corrupted data in preRestore hook
				By("Restoring the backed up data")
				restoreSession, err := f.SetupDatabaseRestore(appBinding, repo, func(restore *v1beta1.RestoreSession) {
					restore.Spec.Target.Alias = ""
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession has succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

				By("Verifying that the original data has been restored")
				res, err = f.ReadProperty(db, sampleTable, property)
				Expect(err).NotTo(HaveOccurred())
				Expect(res).Should(BeEquivalentTo(originalValue))
			})
		})
	})

	Context("Deprecated '.spec.rules' section of RestoreSession", func() {
		It("should move the rules into `spec.target.rules` section", func() {
			// Deploy a Deployment
			deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())

			// Generate Sample Data
			sampleData, err := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Setup a Minio Repository
			repo, err := f.SetupMinioRepository()
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
			rules := []v1beta1.Rule{
				{
					Paths: []string{framework.TestSourceDataMountPath},
				},
			}
			restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment, framework.SourceVolume, func(restore *v1beta1.RestoreSession) {
				restore.Spec.Rules = rules
				restore.Spec.Target.Rules = nil
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that RestoreSession succeeded")
			completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

			By("Verifying that rules has been moved into appropriate position")
			Expect(framework.RulesMigrated(completedRS, rules)).Should(BeTrue())

			// Get restored data
			restoredData := f.RestoredData(deployment.ObjectMeta, apis.KindDeployment)

			// Verify that restored data is same as the original data
			By("Verifying restored data is same as the original data")
			Expect(restoredData).Should(BeSameAs(sampleData))

		})
	})
})
