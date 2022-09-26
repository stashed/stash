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
	"fmt"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gomodules.xyz/x/crypto/rand"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, v1beta1.BackupTargetFound).Should(BeEquivalentTo(core.ConditionFalse))

			By("Creating Deployment")
			deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume, func(dp *apps.Deployment) {
				dp.Name = dpMeta.Name
			})
			Expect(err).NotTo(HaveOccurred())

			// Generate Sample Data
			sampleData, err := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that BackupTargetFound condition is 'True'")
			f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, v1beta1.BackupTargetFound).Should(BeEquivalentTo(core.ConditionTrue))

			By("Checking that StashSidecarInjected condition is 'True'")
			f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, v1beta1.StashSidecarInjected).Should(BeEquivalentTo(core.ConditionTrue))

			By("Checking that CronJobCreated condition is 'True'")
			f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, v1beta1.CronJobCreated).Should(BeEquivalentTo(core.ConditionTrue))

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
			f.EventuallyCondition(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession, v1beta1.RestoreTargetFound).Should(BeEquivalentTo(core.ConditionFalse))

			// Deploy restored Deployment
			restoredDeployment, err := f.DeployDeployment(framework.RestoredDeployment, int32(1), framework.RestoredVolume, func(dp *apps.Deployment) {
				dp.ObjectMeta = rdpMeta
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that RestoreTargetFound condition is 'True'")
			f.EventuallyCondition(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession, v1beta1.RestoreTargetFound).Should(BeEquivalentTo(core.ConditionTrue))

			By("Checking that StashInitContainerInjected condition is 'True'")
			f.EventuallyCondition(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession, v1beta1.StashInitContainerInjected).Should(BeEquivalentTo(core.ConditionTrue))

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
			f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, v1beta1.BackupTargetFound).Should(BeEquivalentTo(core.ConditionFalse))

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
			f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, v1beta1.BackupTargetFound).Should(BeEquivalentTo(core.ConditionTrue))

			By("Checking that CronJobCreated condition is 'True'")
			f.EventuallyCondition(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration, v1beta1.CronJobCreated).Should(BeEquivalentTo(core.ConditionTrue))

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
			f.EventuallyCondition(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession, v1beta1.RestoreTargetFound).Should(BeEquivalentTo(core.ConditionFalse))

			// Create restored PVC
			restoredPVC, err := f.CreateNewPVC(rpvcMeta.Name)
			Expect(err).NotTo(HaveOccurred())

			// Deploy another Pod
			restoredPod, err := f.DeployPod(restoredPVC.Name)
			Expect(err).NotTo(HaveOccurred())

			By("Checking that RestoreTargetFound condition is 'True'")
			f.EventuallyCondition(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession, v1beta1.RestoreTargetFound).Should(BeEquivalentTo(core.ConditionTrue))

			By("Checking that RestoreExecutorEnsured condition is 'True'")
			f.EventuallyCondition(restoreSession.ObjectMeta, v1beta1.ResourceKindRestoreSession, v1beta1.RestoreExecutorEnsured).Should(BeEquivalentTo(core.ConditionTrue))

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
})
