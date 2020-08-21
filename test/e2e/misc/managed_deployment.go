/*
Copyright AppsCode Inc. and Contributors

Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md

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

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kmapi "kmodules.xyz/client-go/api/v1"
	pfutil "kmodules.xyz/client-go/tools/portforward"
	appcatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
)

const (
	sampleTable = "stashDemo"
)

var _ = Describe("Managed Deployment", func() {

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

	Context("Workload", func() {
		It("should wait for the workload", func() {
			dpMeta := metav1.ObjectMeta{
				Name:      rand.WithUniqSuffix(fmt.Sprintf("%s-%s", framework.SourceDeployment, f.App())),
				Namespace: f.Namespace(),
			}

			// Setup a Minio Repository
			repo, err := f.SetupMinioRepository()
			Expect(err).NotTo(HaveOccurred())

			By("Creating BackupConfiguration")
			backupConfig, err := f.CreateBackupConfigForWorkload(dpMeta, repo, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that BackupTargetFound condition is 'False'")
			f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, apis.BackupTargetFound).Should(BeEquivalentTo(kmapi.ConditionFalse))

			By("Creating Deployment")
			deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume, func(dp *apps.Deployment) {
				dp.Name = dpMeta.Name
			})
			Expect(err).NotTo(HaveOccurred())

			// Generate Sample Data
			sampleData, err := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that BackupTargetFound condition is 'True'")
			f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, apis.BackupTargetFound).Should(BeEquivalentTo(kmapi.ConditionTrue))

			By("Checking that StashSidecarInjected condition is 'True'")
			f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, apis.StashSidecarInjected).Should(BeEquivalentTo(kmapi.ConditionTrue))

			By("Checking that CronJobCreated condition is 'True'")
			f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, apis.CronJobCreated).Should(BeEquivalentTo(kmapi.ConditionTrue))

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

			rdpMeta := metav1.ObjectMeta{
				Name:      rand.WithUniqSuffix(fmt.Sprintf("%s-%s", framework.RestoredDeployment, f.App())),
				Namespace: f.Namespace(),
			}

			// Create RestoreSession
			restoreSession, err := f.CreateRestoreSessionForWorkload(rdpMeta, repo.Name, apis.KindDeployment, framework.RestoredVolume)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that RestoreTargetFound condition is 'False'")
			f.EventuallyCondition(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession, apis.RestoreTargetFound).Should(BeEquivalentTo(kmapi.ConditionFalse))

			// Deploy restored Deployment
			restoredDeployment, err := f.DeployDeployment(framework.RestoredDeployment, int32(1), framework.RestoredVolume, func(dp *apps.Deployment) {
				dp.ObjectMeta = rdpMeta
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that RestoreTargetFound condition is 'True'")
			f.EventuallyCondition(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession, apis.RestoreTargetFound).Should(BeEquivalentTo(kmapi.ConditionTrue))

			By("Checking that StashInitContainerInjected condition is 'True'")
			f.EventuallyCondition(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession, apis.StashInitContainerInjected).Should(BeEquivalentTo(kmapi.ConditionTrue))

			By("Verifying that RestoreSession succeeded")
			completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

			// Get restored data
			restoredData := f.RestoredData(restoredDeployment.ObjectMeta, apis.KindDeployment)

			// Verify that restored data is same as the original data
			By("Verifying restored data is same as the original data")
			Expect(restoredData).Should(BeSameAs(sampleData))
		})
	})

	Context("PersistentVolumeClaim", func() {
		It("should wait for the PVC", func() {
			pvcMeta := metav1.ObjectMeta{
				Name:      rand.WithUniqSuffix(fmt.Sprintf("%s-%s", framework.SourceVolume, f.App())),
				Namespace: f.Namespace(),
			}

			// Setup a Minio Repository
			repo, err := f.SetupMinioRepository()
			Expect(err).NotTo(HaveOccurred())

			By("Creating BackupConfiguration")
			backupConfig := f.GetBackupConfiguration(repo.Name, func(bc *v1beta1.BackupConfiguration) {
				bc.Spec.Target = &v1beta1.BackupTarget{
					Ref: framework.GetTargetRef(pvcMeta.Name, apis.KindPersistentVolumeClaim),
				}
				bc.Spec.Task.Name = framework.TaskPVCBackup
			})
			backupConfig, err = f.StashClient.StashV1beta1().BackupConfigurations(backupConfig.Namespace).Create(context.TODO(), backupConfig, metav1.CreateOptions{})
			f.AppendToCleanupList(backupConfig)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that BackupTargetFound condition is 'False'")
			f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, apis.BackupTargetFound).Should(BeEquivalentTo(kmapi.ConditionFalse))

			// Create PVC
			pvc, err := f.CreateNewPVC(pvcMeta.Name)
			Expect(err).NotTo(HaveOccurred())

			// Deploy a Pod
			pod, err := f.DeployPod(pvc.Name)
			Expect(err).NotTo(HaveOccurred())

			// Generate Sample Data
			sampleData, err := f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that BackupTargetFound condition is 'True'")
			f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, apis.BackupTargetFound).Should(BeEquivalentTo(kmapi.ConditionTrue))

			By("Checking that CronJobCreated condition is 'True'")
			f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, apis.CronJobCreated).Should(BeEquivalentTo(kmapi.ConditionTrue))

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

			rpvcMeta := metav1.ObjectMeta{
				Name:      rand.WithUniqSuffix(fmt.Sprintf("%s-%s", framework.RestoredVolume, f.App())),
				Namespace: f.Namespace(),
			}

			By("Creating RestoreSession")
			restoreSession := f.GetRestoreSession(repo.Name, func(restore *v1beta1.RestoreSession) {
				restore.Spec.Target = &v1beta1.RestoreTarget{
					Alias: f.App(),
					Ref:   framework.GetTargetRef(rpvcMeta.Name, apis.KindPersistentVolumeClaim),
					Rules: []v1beta1.Rule{
						{
							Snapshots: []string{"latest"},
						},
					},
				}
				restore.Spec.Task.Name = framework.TaskPVCRestore
			})
			restoreSession, err = f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Create(context.TODO(), restoreSession, metav1.CreateOptions{})
			f.AppendToCleanupList(restoreSession)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that RestoreTargetFound condition is 'False'")
			f.EventuallyCondition(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession, apis.RestoreTargetFound).Should(BeEquivalentTo(kmapi.ConditionFalse))

			// Create restored PVC
			restoredPVC, err := f.CreateNewPVC(rpvcMeta.Name)
			Expect(err).NotTo(HaveOccurred())

			// Deploy another Pod
			restoredPod, err := f.DeployPod(restoredPVC.Name)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that RestoreTargetFound condition is 'True'")
			f.EventuallyCondition(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession, apis.RestoreTargetFound).Should(BeEquivalentTo(kmapi.ConditionTrue))

			By("Checking that RestoreJobCreated condition is 'True'")
			f.EventuallyCondition(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession, apis.RestoreJobCreated).Should(BeEquivalentTo(kmapi.ConditionTrue))

			By("Waiting for restore process to complete")
			f.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession).Should(BeTrue())

			By("Verifying that RestoreSession succeeded")
			completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

			// Get restored data
			restoredData := f.RestoredData(restoredPod.ObjectMeta, apis.KindPod)

			// Verify that restored data is same as the original data
			By("Verifying restored data is same as the original data")
			Expect(restoredData).Should(BeSameAs(sampleData))
		})
	})

	Context("Database", func() {
		Context("MySQL", func() {
			It("should wait for the AppBinding", func() {

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())

				// Prepare MySQL resources
				cred, pvc, svc, dpl, err := f.PrepareMySQLResources(framework.PrefixSource)
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(dpl, svc, pvc, cred)

				appBinding := f.MySQLAppBinding(cred, svc, framework.PrefixSource)
				f.AppendToCleanupList(appBinding)

				By("Creating BackupConfiguration")
				backupConfig := f.GetBackupConfiguration(repo.Name, func(bc *v1beta1.BackupConfiguration) {
					bc.Spec.Task = v1beta1.TaskRef{
						Name: framework.MySQLBackupTask,
					}
					bc.Spec.Target = &v1beta1.BackupTarget{
						Ref: v1beta1.TargetRef{
							APIVersion: appcatalog.SchemeGroupVersion.String(),
							Kind:       apis.KindAppBinding,
							Name:       appBinding.Name,
						},
					}
				})
				backupConfig, err = f.StashClient.StashV1beta1().BackupConfigurations(backupConfig.Namespace).Create(context.TODO(), backupConfig, metav1.CreateOptions{})
				f.AppendToCleanupList(backupConfig)
				Expect(err).NotTo(HaveOccurred())

				By("Checking that BackupTargetFound condition is 'False'")
				f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, apis.BackupTargetFound).Should(BeEquivalentTo(kmapi.ConditionFalse))

				// Create MySQL database
				err = f.CreateMySQL(dpl)
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
				Expect(err).NotTo(HaveOccurred())

				By("Creating AppBinding")
				appBinding, err = f.CreateAppBinding(appBinding)
				Expect(err).NotTo(HaveOccurred())

				By("Checking that BackupTargetFound condition is 'True'")
				f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, apis.BackupTargetFound).Should(BeEquivalentTo(kmapi.ConditionTrue))

				By("Checking that CronJobCreated condition is 'True'")
				f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, apis.CronJobCreated).Should(BeEquivalentTo(kmapi.ConditionTrue))

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

				// Prepare restored MySQL resources
				rcred, rpvc, rsvc, rdpl, err := f.PrepareMySQLResources(framework.PrefixRestore)
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(rdpl, rsvc, rpvc, rcred)

				rappBinding := f.MySQLAppBinding(rcred, rsvc, framework.PrefixRestore)
				f.AppendToCleanupList(rappBinding)

				By("Creating RestoreSession")
				restoreSession := f.GetRestoreSession(repo.Name, func(rs *v1beta1.RestoreSession) {
					rs.Spec.Task = v1beta1.TaskRef{
						Name: framework.MySQLRestoreTask,
					}
					rs.Spec.Target = &v1beta1.RestoreTarget{
						Ref: v1beta1.TargetRef{
							APIVersion: appcatalog.SchemeGroupVersion.String(),
							Kind:       apis.KindAppBinding,
							Name:       rappBinding.Name,
						},
					}
				})
				restoreSession, err = f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Create(context.TODO(), restoreSession, metav1.CreateOptions{})
				f.AppendToCleanupList(restoreSession)
				Expect(err).NotTo(HaveOccurred())

				By("Checking that RestoreTargetFound condition is 'False'")
				f.EventuallyCondition(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession, apis.RestoreTargetFound).Should(BeEquivalentTo(kmapi.ConditionFalse))

				// Create MySQL database
				err = f.CreateMySQL(rdpl)
				Expect(err).NotTo(HaveOccurred())

				By("Creating AppBinding")
				rappBinding, err = f.CreateAppBinding(rappBinding)
				Expect(err).NotTo(HaveOccurred())

				By("Checking that RestoreTargetFound condition is 'True'")
				f.EventuallyCondition(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession, apis.RestoreTargetFound).Should(BeEquivalentTo(kmapi.ConditionTrue))

				By("Checking that RestoreJobCreated condition is 'True'")
				f.EventuallyCondition(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession, apis.RestoreJobCreated).Should(BeEquivalentTo(kmapi.ConditionTrue))

				By("Waiting for restore process to complete")
				f.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession).Should(BeTrue())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

				By("Port forwarding MySQL pod")
				pod, err = f.GetPod(rdpl.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				rtunnel := pfutil.NewTunnel(f.KubeClient.CoreV1().RESTClient(), f.ClientConfig, pod.Namespace, pod.Name, framework.MySQLServingPortNumber)
				defer rtunnel.Close()
				err = rtunnel.ForwardPort()
				Expect(err).NotTo(HaveOccurred())

				By("Connecting with MySQL Server")
				connstr = fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql", framework.SuperUser, f.App(), framework.LocalHostIP, rtunnel.Local)
				rdb, err := sql.Open("mysql", connstr)
				Expect(err).NotTo(HaveOccurred())
				defer rdb.Close()
				rdb.SetConnMaxLifetime(time.Second * 10)
				err = f.EventuallyConnectWithMySQLServer(rdb)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the sample table has been restored")
				tables, err = f.ListTables(rdb)
				Expect(err).NotTo(HaveOccurred())
				Expect(tables.Has(sampleTable)).Should(BeTrue())
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
