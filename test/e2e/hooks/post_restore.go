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

package hooks

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	app_util "kmodules.xyz/client-go/apps/v1"
	pfutil "kmodules.xyz/client-go/tools/portforward"
	probev1 "kmodules.xyz/prober/api/v1"
)

var _ = Describe("PostRestore Hook", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
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
	})

	Context("ExecAction", func() {
		Context("Sidecar Model", func() {
			Context("Success Test", func() {
				It("should execute the postRestore hook successfully", func() {
					// Deploy a StatefulSet with prober client. Here, we are using a StatefulSet because we need a stable address
					// for pod where http request will be sent.
					statefulset, err := f.DeployStatefulSetWithProbeClient()
					Expect(err).NotTo(HaveOccurred())

					// Read data at empty state
					emptyData, err := f.ReadSampleDataFromFromWorkload(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())

					// Generate Sample Data
					sampleData, err := f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())
					Expect(sampleData).ShouldNot(BeSameAs(emptyData))

					// Setup a Minio Repository
					repo, err := f.SetupMinioRepository()
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(repo)

					// Setup workload Backup
					backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup of the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has succeeded")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

					// Simulate disaster scenario. Remove the old data. Then add a demo corrupted file.
					// This corrupted file will be deleted in postRestore hook.
					By("Modifying source data")
					pod, err := f.GetPod(statefulset.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					_, err = f.ExecOnPod(pod, "/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", framework.TestSourceDataMountPath))
					Expect(err).NotTo(HaveOccurred())
					_, err = f.ExecOnPod(pod, "touch", filepath.Join(framework.TestSourceDataMountPath, "corrupted-data.txt"))
					Expect(err).NotTo(HaveOccurred())

					// Restore the backed up data
					// Remove the corrupted data in postRestore hook.
					By("Restoring the backed up data in the original StatefulSet")
					restoreSession, err := f.SetupRestoreProcess(statefulset.ObjectMeta, repo, apis.KindStatefulSet, framework.SourceVolume, func(restore *v1beta1.RestoreSession) {
						restore.Spec.Hooks = &v1beta1.RestoreHooks{
							PostRestore: &probev1.Handler{
								Exec: &core.ExecAction{
									Command: []string{"/bin/sh", "-c", fmt.Sprintf("rm %s/corrupted-data.txt", framework.TestSourceDataMountPath)},
								},
								ContainerName: util.StashInitContainer,
							},
						}
					})
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that RestoreSession succeeded")
					completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

					restoredData := f.RestoredData(statefulset.ObjectMeta, apis.KindStatefulSet)
					By("Verifying that the original data has been restored and corrupted file has been removed")
					Expect(restoredData).Should(BeSameAs(sampleData))
				})

				It("should execute the postRestore hook even when the restore process failed", func() {
					// Deploy a StatefulSet with prober client. Here, we are using a StatefulSet because we need a stable address
					// for pod where http request will be sent.
					statefulset, err := f.DeployStatefulSetWithProbeClient()
					Expect(err).NotTo(HaveOccurred())

					// Read data at empty state
					emptyData, err := f.ReadSampleDataFromFromWorkload(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())

					// Generate Sample Data
					sampleData, err := f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())
					Expect(sampleData).ShouldNot(BeSameAs(emptyData))

					// Setup a Minio Repository
					repo, err := f.SetupMinioRepository()
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(repo)

					// Setup workload Backup
					backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup of the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has succeeded")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

					// Simulate disaster scenario. Remove the old data. Then add a demo corrupted file.
					// This corrupted file will be deleted in postRestore hook.
					By("Modifying source data")
					pod, err := f.GetPod(statefulset.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					_, err = f.ExecOnPod(pod, "/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", framework.TestSourceDataMountPath))
					Expect(err).NotTo(HaveOccurred())
					_, err = f.ExecOnPod(pod, "touch", filepath.Join(framework.TestSourceDataMountPath, "corrupted-data.txt"))
					Expect(err).NotTo(HaveOccurred())

					// Restore the backed up data
					// Try to restore a directory that hasn't been backed up. This will force the restore process to fail.
					// Remove the corrupted data in postRestore hook.
					By("Restoring the backed up data in the original StatefulSet")
					restoreSession, err := f.SetupRestoreProcess(statefulset.ObjectMeta, repo, apis.KindStatefulSet, framework.SourceVolume, func(restore *v1beta1.RestoreSession) {
						restore.Spec.Hooks = &v1beta1.RestoreHooks{
							PostRestore: &probev1.Handler{
								Exec: &core.ExecAction{
									Command: []string{"/bin/sh", "-c", fmt.Sprintf("rm %s/corrupted-data.txt", framework.TestSourceDataMountPath)},
								},
								ContainerName: util.StashInitContainer,
							},
						}
						restore.Spec.Rules = []v1beta1.Rule{
							{
								Paths: []string{"/unknown/directory"},
							},
						}
					})
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that RestoreSession has failed")
					completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionFailed))

					// Delete RestoreSession so that the StatefulSet can start normally
					By("Deleting RestoreSession")
					err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					// delete failed pod so that StatefulSet can start
					err = f.DeletePod(pod.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					err = app_util.WaitUntilStatefulSetReady(f.KubeClient, statefulset.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					restoredData := f.RestoredData(statefulset.ObjectMeta, apis.KindStatefulSet)
					By("Verifying that the corrupted file has been removed")
					Expect(restoredData).Should(BeSameAs(emptyData))
				})
			})

			Context("Failure Test", func() {
				It("should restore backed up data even when the hook failed", func() {
					// Deploy a StatefulSet with prober client. Here, we are using a StatefulSet because we need a stable address
					// for pod where http request will be sent.
					statefulset, err := f.DeployStatefulSetWithProbeClient()
					Expect(err).NotTo(HaveOccurred())

					// Read data at empty state
					emptyData, err := f.ReadSampleDataFromFromWorkload(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())

					// Generate Sample Data
					sampleData, err := f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())
					Expect(sampleData).ShouldNot(BeSameAs(emptyData))

					// Setup a Minio Repository
					repo, err := f.SetupMinioRepository()
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(repo)

					// Setup workload Backup
					backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup of the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has succeeded")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

					// Simulate disaster scenario. Remove old data
					By("Removing source data")
					pod, err := f.GetPod(statefulset.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					_, err = f.ExecOnPod(pod, "/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", framework.TestSourceDataMountPath))
					Expect(err).NotTo(HaveOccurred())

					// Restore the backed up data
					// Return non-zero exit code from postRestore hook so that it fail
					By("Restoring the backed up data in the original StatefulSet")
					restoreSession, err := f.SetupRestoreProcess(statefulset.ObjectMeta, repo, apis.KindStatefulSet, framework.SourceVolume, func(restore *v1beta1.RestoreSession) {
						restore.Spec.Hooks = &v1beta1.RestoreHooks{
							PostRestore: &probev1.Handler{
								Exec: &core.ExecAction{
									Command: []string{"/bin/sh", "-c", "exit 1"},
								},
								ContainerName: util.StashInitContainer,
							},
						}
					})
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that RestoreSession has failed")
					completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionFailed))

					// Delete RestoreSession so that the StatefulSet can start normally
					By("Deleting RestoreSession")
					err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					// delete failed pod so that StatefulSet can start
					err = f.DeletePod(pod.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					err = app_util.WaitUntilStatefulSetReady(f.KubeClient, statefulset.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					restoredData := f.RestoredData(statefulset.ObjectMeta, apis.KindStatefulSet)
					By("Verifying that data has been restored")
					Expect(restoredData).Should(BeSameAs(sampleData))
				})
			})
		})

		Context("Job Model", func() {
			Context("PVC", func() {
				Context("Success Cases", func() {
					It("should execute postRestore hook successfully", func() {
						// Create new PVC
						pvc, err := f.CreateNewPVC(fmt.Sprintf("%s-%s", framework.SourceVolume, f.App()))
						Expect(err).NotTo(HaveOccurred())

						// Deploy a Pod
						pod, err := f.DeployPod(pvc.Name)
						Expect(err).NotTo(HaveOccurred())

						// Read data at empty state
						emptyData, err := f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())

						// Generate Sample Data
						sampleData, err := f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())
						Expect(sampleData).ShouldNot(BeSameAs(emptyData))

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						// Setup PVC Backup
						backupConfig, err := f.SetupPVCBackup(pvc, repo)
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has succeeded")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

						// Simulate disaster scenario. Remove the old data. Then add a demo corrupted file.
						// This corrupted file will be deleted in postRestore hook.
						By("Modifying source data")
						_, err = f.ExecOnPod(pod, "/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", framework.TestSourceDataMountPath))
						Expect(err).NotTo(HaveOccurred())
						_, err = f.ExecOnPod(pod, "touch", filepath.Join(framework.TestSourceDataMountPath, "corrupted-data.txt"))
						Expect(err).NotTo(HaveOccurred())

						// Restore the backed up data
						// Cleanup corrupted data in postRestore hook
						By("Restoring the backed up data")
						restoreSession, err := f.SetupRestoreProcessForPVC(pvc, repo, func(restore *v1beta1.RestoreSession) {
							restore.Spec.Hooks = &v1beta1.RestoreHooks{
								PostRestore: &probev1.Handler{
									Exec: &core.ExecAction{
										Command: []string{"/bin/sh", "-c", fmt.Sprintf("rm %s/corrupted-data.txt", apis.StashDefaultMountPath)},
									},
									ContainerName: apis.PostTaskHook,
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that RestoreSession has succeeded")
						completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

						restoredData := f.RestoredData(pod.ObjectMeta, apis.KindPod)
						By("Verifying that the original data has been restored and corrupted file has been removed")
						Expect(restoredData).Should(BeSameAs(sampleData))
					})

					It("should execute postRestore hook even when the restore process failed", func() {
						// Create new PVC
						pvc, err := f.CreateNewPVC(fmt.Sprintf("%s-%s", framework.SourceVolume, f.App()))
						Expect(err).NotTo(HaveOccurred())

						// Deploy a Pod
						pod, err := f.DeployPod(pvc.Name)
						Expect(err).NotTo(HaveOccurred())

						// Read data at empty state
						emptyData, err := f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())

						// Generate Sample Data
						sampleData, err := f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())
						Expect(sampleData).ShouldNot(BeSameAs(emptyData))

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						// Setup PVC Backup
						backupConfig, err := f.SetupPVCBackup(pvc, repo)
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has succeeded")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

						// Simulate disaster scenario. Remove the old data. Then add a demo corrupted file.
						// This corrupted file will be deleted in postRestore hook.
						By("Modifying source data")
						_, err = f.ExecOnPod(pod, "/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", framework.TestSourceDataMountPath))
						Expect(err).NotTo(HaveOccurred())
						_, err = f.ExecOnPod(pod, "touch", filepath.Join(framework.TestSourceDataMountPath, "corrupted-data.txt"))
						Expect(err).NotTo(HaveOccurred())

						// Restore the backed up data
						// Try to restore an invalid snapshot so that the restore process fail
						// Cleanup corrupted data in postRestore hook
						By("Restoring the backed up data")
						restoreSession, err := f.SetupRestoreProcessForPVC(pvc, repo, func(restore *v1beta1.RestoreSession) {
							restore.Spec.Hooks = &v1beta1.RestoreHooks{
								PostRestore: &probev1.Handler{
									Exec: &core.ExecAction{
										Command: []string{"/bin/sh", "-c", fmt.Sprintf("rm %s/corrupted-data.txt", apis.StashDefaultMountPath)},
									},
									ContainerName: apis.PostTaskHook,
								},
							}
							restore.Spec.Rules = []v1beta1.Rule{
								{
									Snapshots: []string{"invalid-snapshot"},
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that RestoreSession has failed")
						completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionFailed))

						restoredData := f.RestoredData(pod.ObjectMeta, apis.KindPod)
						By("Verifying that the corrupted file has been removed")
						Expect(restoredData).Should(BeSameAs(emptyData))
					})
				})

				Context("Failure Cases", func() {
					It("should restore the backed up data even when the hook failed", func() {
						// Create new PVC
						pvc, err := f.CreateNewPVC(fmt.Sprintf("%s-%s", framework.SourceVolume, f.App()))
						Expect(err).NotTo(HaveOccurred())

						// Deploy a Pod
						pod, err := f.DeployPod(pvc.Name)
						Expect(err).NotTo(HaveOccurred())

						// Read data at empty state
						emptyData, err := f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())

						// Generate Sample Data
						sampleData, err := f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())
						Expect(sampleData).ShouldNot(BeSameAs(emptyData))

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						// Setup PVC Backup
						backupConfig, err := f.SetupPVCBackup(pvc, repo)
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has succeeded")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

						// Simulate disaster scenario. Remove old data
						By("Removing source data")
						_, err = f.ExecOnPod(pod, "/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", framework.TestSourceDataMountPath))
						Expect(err).NotTo(HaveOccurred())

						// Restore the backed up data
						// Return non-zero exit code from postRestore hook so that the hook fail
						By("Restoring the backed up data")
						restoreSession, err := f.SetupRestoreProcessForPVC(pvc, repo, func(restore *v1beta1.RestoreSession) {
							restore.Spec.Hooks = &v1beta1.RestoreHooks{
								PostRestore: &probev1.Handler{
									Exec: &core.ExecAction{
										Command: []string{"/bin/sh", "-c", "exit 1"},
									},
									ContainerName: apis.PostTaskHook,
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that the RestoreSession has failed")
						completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionFailed))

						restoredData := f.RestoredData(pod.ObjectMeta, apis.KindPod)
						By("Verifying that the sample data has been restored")
						Expect(restoredData).Should(BeSameAs(sampleData))
					})
				})
			})

			Context("MySQL", func() {
				const (
					sampleTable   = "StashDemo"
					property      = "price"
					originalValue = 123456
				)

				BeforeEach(func() {
					// Skip test if respective Functions and Tasks are not installed.
					if !f.MySQLAddonInstalled() {
						Skip("MySQL addon is not installed")
					}
				})

				Context("Success Test", func() {
					It("should execute postRestore hook successfully", func() {
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
						backupConfig, err := f.SetupDatabaseBackup(appBinding, repo)
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has succeeded")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
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
						// Insert a restoreCount row in postRestore hook that will be used to verify
						// whether the postRestore hook executed or not.
						By("Restoring the backed up data")
						propertyRestoreCount := "restore_count"
						restoreCount := 1
						restoreSession, err := f.SetupDatabaseRestore(appBinding, repo, func(restore *v1beta1.RestoreSession) {
							restore.Spec.Hooks = &v1beta1.RestoreHooks{
								PostRestore: &probev1.Handler{
									Exec: &core.ExecAction{
										Command: []string{"/bin/sh", "-c",
											fmt.Sprintf("`mysql -u root --password=$MYSQL_ROOT_PASSWORD -e \"USE mysql; INSERT INTO %s(property, value) VALUES('%s', %d);\"`", sampleTable, propertyRestoreCount, restoreCount)},
									},
									ContainerName: framework.MySQLContainerName,
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that RestoreSession has succeeded")
						completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

						By("Verifying that the original data has been restored")
						res, err = f.ReadProperty(db, sampleTable, property)
						Expect(err).NotTo(HaveOccurred())
						Expect(res).Should(BeEquivalentTo(originalValue))

						By("Verifying that postRestore hook has been executed")
						res, err = f.ReadProperty(db, sampleTable, propertyRestoreCount)
						Expect(err).NotTo(HaveOccurred())
						Expect(res).Should(BeEquivalentTo(restoreCount))
					})

					It("should execute postRestore hook even when restore process failed", func() {
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
						backupConfig, err := f.SetupDatabaseBackup(appBinding, repo)
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has succeeded")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
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
						// Try to restore an invalid snapshot so that the restore process fail
						// Insert a restoreCount row in postRestore hook that will be used to verify
						// whether the postRestore hook executed or not.
						By("Restoring the backed up data")
						propertyRestoreCount := "restore_count"
						restoreCount := 1
						restoreSession, err := f.SetupDatabaseRestore(appBinding, repo, func(restore *v1beta1.RestoreSession) {
							restore.Spec.Hooks = &v1beta1.RestoreHooks{
								PostRestore: &probev1.Handler{
									Exec: &core.ExecAction{
										Command: []string{"/bin/sh", "-c",
											fmt.Sprintf("`mysql -u root --password=$MYSQL_ROOT_PASSWORD -e \"USE mysql; INSERT INTO %s(property, value) VALUES('%s', %d);\"`", sampleTable, propertyRestoreCount, restoreCount)},
									},
									ContainerName: framework.MySQLContainerName,
								},
							}
							restore.Spec.Rules = []v1beta1.Rule{
								{
									Snapshots: []string{"invalid-snapshot"},
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that RestoreSession has failed")
						completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionFailed))

						By("Verifying that the table contain corrupted data")
						res, err = f.ReadProperty(db, sampleTable, property)
						Expect(err).NotTo(HaveOccurred())
						Expect(res).ShouldNot(BeEquivalentTo(originalValue))

						By("Verifying that postRestore hook has been executed")
						res, err = f.ReadProperty(db, sampleTable, propertyRestoreCount)
						Expect(err).NotTo(HaveOccurred())
						Expect(res).Should(BeEquivalentTo(restoreCount))
					})
				})

				Context("Failure Test", func() {
					It("should restore even when the postRestore hook failed", func() {
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
						backupConfig, err := f.SetupDatabaseBackup(appBinding, repo)
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has succeeded")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
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
						// Return non-zero return code from the hook to fail it.
						By("Restoring the backed up data")
						restoreSession, err := f.SetupDatabaseRestore(appBinding, repo, func(restore *v1beta1.RestoreSession) {
							restore.Spec.Hooks = &v1beta1.RestoreHooks{
								PostRestore: &probev1.Handler{
									Exec: &core.ExecAction{
										Command: []string{"/bin/sh", "-c", "exit 1"},
									},
									ContainerName: framework.MySQLContainerName,
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that RestoreSession has failed")
						completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionFailed))

						By("Verifying that the original data has been restored")
						res, err = f.ReadProperty(db, sampleTable, property)
						Expect(err).NotTo(HaveOccurred())
						Expect(res).Should(BeEquivalentTo(originalValue))
					})
				})
			})
		})
	})
})
