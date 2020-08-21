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
	"fmt"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

var _ = Describe("Runtime Settings", func() {

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

	Context("NICE and IONICE", func() {
		Context("Sidecar Model", func() {
			It("should Backup & Restore successfully", func() {
				// Deploy a Deployment
				deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
					bc.Spec.RuntimeSettings = ofst.RuntimeSettings{
						Container: &ofst.ContainerRuntimeSettings{
							Nice: &ofst.NiceSettings{
								Adjustment: types.Int32P(5),
							},
							IONice: &ofst.IONiceSettings{
								Class:     types.Int32P(2),
								ClassData: types.Int32P(4),
							},
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
				restoreSession, err := f.SetupRestoreProcess(deployment.ObjectMeta, repo, apis.KindDeployment, framework.SourceVolume, func(restore *v1beta1.RestoreSession) {
					restore.Spec.RuntimeSettings = ofst.RuntimeSettings{
						Container: &ofst.ContainerRuntimeSettings{
							Nice: &ofst.NiceSettings{
								Adjustment: types.Int32P(5),
							},
							IONice: &ofst.IONiceSettings{
								Class:     types.Int32P(2),
								ClassData: types.Int32P(4),
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
				restoredData := f.RestoredData(deployment.ObjectMeta, apis.KindDeployment)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})
	})

	Context("Pod runtimeSettings", func() {
		Context("Sidecar Model", func() {
			It("should not be applied on the workloads", func() {
				// Deploy a Deployment
				deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
					bc.Spec.RuntimeSettings = ofst.RuntimeSettings{
						Pod: &ofst.PodRuntimeSettings{
							SecurityContext: &core.PodSecurityContext{
								FSGroup: types.Int64P(framework.TestFSGroup),
							},
						},
					}
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that runtimeSettings has been applied on the CronJob")
				cronJob, err := f.GetCronJob(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				Expect(framework.HasFSGroup(cronJob.Spec.JobTemplate.Spec.Template.Spec.SecurityContext)).Should(BeTrue())

				By("Verifying that runtimeSettings hasn't been applied on the workload")
				dpl, err := f.KubeClient.AppsV1().Deployments(deployment.Namespace).Get(context.TODO(), deployment.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(framework.HasFSGroup(dpl.Spec.Template.Spec.SecurityContext)).ShouldNot(BeTrue())

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

				// Deploy restored Deployment
				restoredDeployment, err := f.DeployDeployment(framework.RestoredDeployment, int32(1), framework.RestoredVolume)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data in different Deployment")
				restoreSession, err := f.SetupRestoreProcess(restoredDeployment.ObjectMeta, repo, apis.KindDeployment, framework.RestoredVolume, func(restore *v1beta1.RestoreSession) {
					restore.Spec.RuntimeSettings = ofst.RuntimeSettings{
						Pod: &ofst.PodRuntimeSettings{
							SecurityContext: &core.PodSecurityContext{
								FSGroup: types.Int64P(framework.TestFSGroup),
							},
						},
					}
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the runtimeSettings hasn't been applied on the restored Deployment")
				dpl, err = f.KubeClient.AppsV1().Deployments(deployment.Namespace).Get(context.TODO(), restoredDeployment.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(framework.HasFSGroup(dpl.Spec.Template.Spec.SecurityContext)).ShouldNot(BeTrue())

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

		Context("Job Model", func() {
			It("should apply runtimeSettings on the backup & restore job", func() {
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
					bc.Spec.RuntimeSettings = ofst.RuntimeSettings{
						Pod: &ofst.PodRuntimeSettings{
							SecurityContext: &core.PodSecurityContext{
								FSGroup: types.Int64P(framework.TestFSGroup),
							},
						},
					}
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that runtimeSettings has been applied on the CronJob")
				cronJob, err := f.GetCronJob(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				Expect(framework.HasFSGroup(cronJob.Spec.JobTemplate.Spec.Template.Spec.SecurityContext)).Should(BeTrue())

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

				By("Verifying that the runtimeSettings has been applied on the backup job")
				job, err := f.GetBackupJob(backupSession.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(framework.HasFSGroup(job.Spec.Template.Spec.SecurityContext)).Should(BeTrue())

				// Create restored Pvc
				restoredPVC, err := f.CreateNewPVC(fmt.Sprintf("%s-%s", framework.RestoredVolume, f.App()))
				Expect(err).NotTo(HaveOccurred())

				// Deploy another Pod
				restoredPod, err := f.DeployPod(restoredPVC.Name)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data")
				restoreSession, err := f.SetupRestoreProcessForPVC(restoredPVC, repo, func(restore *v1beta1.RestoreSession) {
					restore.Spec.RuntimeSettings = ofst.RuntimeSettings{
						Pod: &ofst.PodRuntimeSettings{
							SecurityContext: &core.PodSecurityContext{
								FSGroup: types.Int64P(framework.TestFSGroup),
							},
						},
					}
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

				By("Verifying that the runtimeSettings has been applied on the restore job")
				jobs, err := f.GetRestoreJobs()
				Expect(err).NotTo(HaveOccurred())
				for i := range jobs {
					if framework.JobsTargetMatch(jobs[i], restoreSession.Spec.Target.Ref) {
						Expect(framework.HasFSGroup(jobs[i].Spec.Template.Spec.SecurityContext)).Should(BeTrue())
					}
				}

				// Get restored data
				restoredData := f.RestoredData(restoredPod.ObjectMeta, apis.KindPod)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})
	})

	Context("Container runtimeSettings", func() {
		Context("Sidecar Model", func() {
			It("should apply on the workloads", func() {
				// Deploy a Deployment
				deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
					bc.Spec.RuntimeSettings = ofst.RuntimeSettings{
						Container: &ofst.ContainerRuntimeSettings{
							Resources: core.ResourceRequirements{
								Limits: core.ResourceList{
									core.ResourceMemory: resource.MustParse(framework.TestResourceLimit),
								},
								Requests: core.ResourceList{
									core.ResourceMemory: resource.MustParse(framework.TestResourceRequest),
								},
							},
							SecurityContext: &core.SecurityContext{
								RunAsUser: types.Int64P(framework.TestUserID),
							},
						},
					}
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that runtimeSettings has been applied on the CronJob")
				cronJob, err := f.GetCronJob(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				Expect(framework.HasResources(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers)).Should(BeTrue())
				Expect(framework.HasSecurityContext(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers)).Should(BeTrue())

				By("Verifying that the runtimeSettings has been applied on the workload")
				dpl, err := f.KubeClient.AppsV1().Deployments(deployment.Namespace).Get(context.TODO(), deployment.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(framework.HasResources(dpl.Spec.Template.Spec.Containers)).Should(BeTrue())
				Expect(framework.HasSecurityContext(dpl.Spec.Template.Spec.Containers)).Should(BeTrue())

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

				// Deploy restored Deployment
				restoredDeployment, err := f.DeployDeployment(framework.RestoredDeployment, int32(1), framework.RestoredVolume)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data in different Deployment")
				restoreSession, err := f.SetupRestoreProcess(restoredDeployment.ObjectMeta, repo, apis.KindDeployment, framework.RestoredVolume, func(restore *v1beta1.RestoreSession) {
					restore.Spec.RuntimeSettings = ofst.RuntimeSettings{
						Container: &ofst.ContainerRuntimeSettings{
							Resources: core.ResourceRequirements{
								Limits: core.ResourceList{
									core.ResourceMemory: resource.MustParse(framework.TestResourceLimit),
								},
								Requests: core.ResourceList{
									core.ResourceMemory: resource.MustParse(framework.TestResourceRequest),
								},
							},
							SecurityContext: &core.SecurityContext{
								RunAsUser: types.Int64P(framework.TestUserID),
							},
						},
					}
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the runtimeSettings has been applied on the restored Deployment")
				dpl, err = f.KubeClient.AppsV1().Deployments(deployment.Namespace).Get(context.TODO(), restoredDeployment.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(framework.HasResources(dpl.Spec.Template.Spec.InitContainers)).Should(BeTrue())
				Expect(framework.HasSecurityContext(dpl.Spec.Template.Spec.InitContainers)).Should(BeTrue())

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

		Context("Job Model", func() {
			It("should apply runtimeSettings on the backup & restore job", func() {
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
					bc.Spec.RuntimeSettings = ofst.RuntimeSettings{
						Container: &ofst.ContainerRuntimeSettings{
							Resources: core.ResourceRequirements{
								Limits: core.ResourceList{
									core.ResourceMemory: resource.MustParse(framework.TestResourceLimit),
								},
								Requests: core.ResourceList{
									core.ResourceMemory: resource.MustParse(framework.TestResourceRequest),
								},
							},
							SecurityContext: &core.SecurityContext{
								RunAsUser: types.Int64P(framework.TestUserID),
							},
						},
					}
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that runtimeSettings has been applied on the CronJob")
				cronJob, err := f.GetCronJob(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				Expect(framework.HasResources(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers)).Should(BeTrue())
				Expect(framework.HasSecurityContext(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers)).Should(BeTrue())

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

				By("Verifying that the runtimeSettings has been applied on the backup job")
				job, err := f.GetBackupJob(backupSession.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(framework.HasResources(append(job.Spec.Template.Spec.InitContainers, job.Spec.Template.Spec.Containers...))).Should(BeTrue())
				Expect(framework.HasSecurityContext(append(job.Spec.Template.Spec.InitContainers, job.Spec.Template.Spec.Containers...))).Should(BeTrue())

				// Create restored Pvc
				restoredPVC, err := f.CreateNewPVC(fmt.Sprintf("%s-%s", framework.RestoredVolume, f.App()))
				Expect(err).NotTo(HaveOccurred())

				// Deploy another Pod
				restoredPod, err := f.DeployPod(restoredPVC.Name)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backed up data
				By("Restoring the backed up data")
				restoreSession, err := f.SetupRestoreProcessForPVC(restoredPVC, repo, func(restore *v1beta1.RestoreSession) {
					restore.Spec.RuntimeSettings = ofst.RuntimeSettings{
						Container: &ofst.ContainerRuntimeSettings{
							Resources: core.ResourceRequirements{
								Limits: core.ResourceList{
									core.ResourceMemory: resource.MustParse(framework.TestResourceLimit),
								},
								Requests: core.ResourceList{
									core.ResourceMemory: resource.MustParse(framework.TestResourceRequest),
								},
							},
							SecurityContext: &core.SecurityContext{
								RunAsUser: types.Int64P(framework.TestUserID),
							},
						},
					}
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

				By("Verifying that the runtimeSettings has been applied on the restore job")
				jobs, err := f.GetRestoreJobs()
				Expect(err).NotTo(HaveOccurred())
				for i := range jobs {
					if framework.JobsTargetMatch(jobs[i], restoreSession.Spec.Target.Ref) {
						Expect(framework.HasResources(append(jobs[i].Spec.Template.Spec.InitContainers, jobs[i].Spec.Template.Spec.Containers...))).Should(BeTrue())
						Expect(framework.HasSecurityContext(append(jobs[i].Spec.Template.Spec.InitContainers, jobs[i].Spec.Template.Spec.Containers...))).Should(BeTrue())
					}
				}

				// Get restored data
				restoredData := f.RestoredData(restoredPod.ObjectMeta, apis.KindPod)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})
	})
})
