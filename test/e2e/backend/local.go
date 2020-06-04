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
	"context"
	"database/sql"
	"fmt"
	"time"

	pfutil "kmodules.xyz/client-go/tools/portforward"
	appcatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
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
		f.PrintDebugInfoOnFailure()
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
				deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
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
				deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
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
		// NFS server tests required nfs-kernel-server to be installed in the kubernetes nodes.
		// If you are using a Kind cluster, install nfs-kernel-server by following commands.
		// $ docker exec kind-control-plane apt-get update
		// $ docker exec kind-control-plane apt-get install -y nfs-kernel-server
		// If the kind cluster has multiple nodes, install nfs-kernel-server in each of them.
		Context("General Backup/Restore", func() {
			It("should backup/restore in/from Local backend", func() {
				// Deploy a Deployment
				deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume, func(dp *apps.Deployment) {
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
					bc.Spec.RuntimeSettings.Container = &ofst.ContainerRuntimeSettings{
						SecurityContext: &core.SecurityContext{
							Privileged: types.BoolP(true),
							RunAsUser:  types.Int64P(int64(0)),
							RunAsGroup: types.Int64P(int64(0)),
						},
					}
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
				restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment, framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
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
				deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume, func(dp *apps.Deployment) {
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
					bc.Spec.RuntimeSettings.Container = &ofst.ContainerRuntimeSettings{
						SecurityContext: &core.SecurityContext{
							Privileged: types.BoolP(true),
							RunAsUser:  types.Int64P(int64(0)),
							RunAsGroup: types.Int64P(int64(0)),
						},
					}
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
				restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment, framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

				// Get restored data
				restoredData := f.RestoredData(deployment.ObjectMeta, apis.KindDeployment)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("BatchBackup", func() {
			const sampleTable = "stashDemo"
			It("should backup all targets", func() {
				var members []v1beta1.BackupConfigurationTemplateSpec

				// Deploy a Deployment and generate sample data inside Deployment
				dpl, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume, func(dp *apps.Deployment) {
					dp.Spec.Template.Spec.Containers[0].SecurityContext = &core.SecurityContext{
						Privileged: types.BoolP(true),
						RunAsUser:  types.Int64P(int64(0)),
						RunAsGroup: types.Int64P(int64(0)),
					}
				})
				Expect(err).NotTo(HaveOccurred())
				_, err = f.GenerateSampleData(dpl.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())
				members = append(members, v1beta1.BackupConfigurationTemplateSpec{
					Task: v1beta1.TaskRef{},
					Target: &v1beta1.BackupTarget{
						Ref: v1beta1.TargetRef{
							APIVersion: apps.SchemeGroupVersion.String(),
							Kind:       apis.KindDeployment,
							Name:       dpl.Name,
						},
						Paths: []string{
							framework.TestSourceDataMountPath,
						},
						VolumeMounts: []core.VolumeMount{
							{
								Name:      framework.SourceVolume,
								MountPath: framework.TestSourceDataMountPath,
							},
						},
					},
					RuntimeSettings: ofst.RuntimeSettings{
						Container: &ofst.ContainerRuntimeSettings{
							SecurityContext: &core.SecurityContext{
								Privileged: types.BoolP(true),
								RunAsUser:  types.Int64P(int64(0)),
								RunAsGroup: types.Int64P(int64(0)),
							},
						},
					},
				})

				// Create new PVC and deploy a Pod to use this pvc
				// then generate sample data inside PVC
				pvc, err := f.CreateNewPVC(fmt.Sprintf("%s-%s", framework.SourcePVC, f.App()))
				Expect(err).NotTo(HaveOccurred())
				pod, err := f.DeployPod(pvc.Name)
				Expect(err).NotTo(HaveOccurred())
				_, err = f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)
				Expect(err).NotTo(HaveOccurred())
				members = append(members, v1beta1.BackupConfigurationTemplateSpec{
					Task: v1beta1.TaskRef{
						Name: framework.TaskPVCBackup,
					},
					Target: &v1beta1.BackupTarget{
						Ref: v1beta1.TargetRef{
							APIVersion: core.SchemeGroupVersion.String(),
							Kind:       apis.KindPersistentVolumeClaim,
							Name:       pvc.Name,
						},
					},
					RuntimeSettings: ofst.RuntimeSettings{
						Container: &ofst.ContainerRuntimeSettings{
							SecurityContext: &core.SecurityContext{
								Privileged: types.BoolP(true),
								RunAsUser:  types.Int64P(int64(0)),
								RunAsGroup: types.Int64P(int64(0)),
							},
						},
					},
				})

				// Setup MySQL Database and generate sample data
				// Deploy MySQL database and respective service,secret,PVC and AppBinding.
				By("Deploying MySQL Server")
				dpl, appBinding, err := f.DeployMySQLDatabase()
				Expect(err).NotTo(HaveOccurred())

				By("Port forwarding MySQL pod")
				pod, err = f.GetPod(dpl.ObjectMeta)
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
				Expect(err).NotTo(HaveOccurred())
				members = append(members, v1beta1.BackupConfigurationTemplateSpec{
					Task: v1beta1.TaskRef{
						Name: framework.MySQLBackupTask,
					},
					Target: &v1beta1.BackupTarget{
						Ref: v1beta1.TargetRef{
							APIVersion: appcatalog.SchemeGroupVersion.String(),
							Kind:       apis.KindAppBinding,
							Name:       appBinding.Name,
						},
					},
					RuntimeSettings: ofst.RuntimeSettings{
						Container: &ofst.ContainerRuntimeSettings{
							SecurityContext: &core.SecurityContext{
								Privileged: types.BoolP(true),
								RunAsUser:  types.Int64P(int64(0)),
								RunAsGroup: types.Int64P(int64(0)),
							},
						},
					},
				})

				// Setup a Local Repository
				repo, err := f.SetupLocalRepositoryWithNFSServer()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup batch backup for all targets
				backupBatch, err := f.SetupBatchBackup(repo, func(in *v1beta1.BackupBatch) {
					in.Spec.Members = members
				})
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupBatch.ObjectMeta, v1beta1.BackupInvokerRef{
					Name: backupBatch.Name,
					Kind: v1beta1.ResourceKindBackupBatch,
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(context.TODO(), backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
			})
		})
	})

	Context("HostPath", func() {
		Context("General Backup/Restore", func() {
			It("should backup/restore in/from Local backend", func() {
				// Deploy a Deployment
				deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Local Repository
				repo, err := f.SetupLocalRepositoryWithHostPath()
				Expect(err).NotTo(HaveOccurred())

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
					bc.Spec.RuntimeSettings.Container = &ofst.ContainerRuntimeSettings{
						SecurityContext: &core.SecurityContext{
							Privileged: types.BoolP(true),
							RunAsUser:  types.Int64P(int64(0)),
							RunAsGroup: types.Int64P(int64(0)),
						},
					}
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
				restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment, framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
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
				deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateBigSampleFile(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Local Repository
				repo, err := f.SetupLocalRepositoryWithPVC()
				Expect(err).NotTo(HaveOccurred())

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
					bc.Spec.RuntimeSettings.Container = &ofst.ContainerRuntimeSettings{
						SecurityContext: &core.SecurityContext{
							Privileged: types.BoolP(true),
							RunAsUser:  types.Int64P(int64(0)),
							RunAsGroup: types.Int64P(int64(0)),
						},
					}
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
				restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment, framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
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
