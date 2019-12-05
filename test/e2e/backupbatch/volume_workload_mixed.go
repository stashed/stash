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

package backupbatch

import (
	"fmt"

	"stash.appscode.dev/stash/apis"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
	"stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Workloads and Volume", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	var (
		setupBackup = func(repo *api.Repository, transformFunc func(in *v1beta1.BackupBatch)) (*v1beta1.BackupBatch, error) {
			// Generate desired BackupConfiguration definition
			backupBatch := f.BackupBatch(repo.Name)
			// transformFunc provide a function that made test specific change on the BackupBatch
			// apply these test specific changes
			transformFunc(backupBatch)

			By("Creating Batch: " + backupBatch.Name)
			createdBackupBatch, err := f.StashClient.StashV1beta1().BackupBatches(backupBatch.Namespace).Create(backupBatch)
			f.AppendToCleanupList(createdBackupBatch)

			By("Verifying that backup triggering CronJob has been created")
			f.EventuallyCronJobCreated(backupBatch.ObjectMeta).Should(BeTrue())

			for _, backupConfigTemp := range backupBatch.Spec.BackupConfigurationTemplates {
				objMeta := metav1.ObjectMeta{
					Namespace: backupBatch.Namespace,
					Name:      backupConfigTemp.Spec.Target.Ref.Name,
				}
				if backupConfigTemp.Spec.Target.Ref.Kind == apis.KindDeployment {
					By("Verifying that sidecar has been injected")
					f.EventuallyDeployment(objMeta).Should(matcher.HaveSidecar(util.StashContainer))
					By("Waiting for Deployment to be ready with sidecar")
					err = f.WaitUntilDeploymentReadyWithSidecar(objMeta)
					if err != nil {
						return createdBackupBatch, err
					}
				}
			}
			return createdBackupBatch, err
		}
	)

	Context("Mixed Backup", func() {

		Context("Backup Workloads and Volume using BackupBatch", func() {
			It("should Backup Workloads and Volume", func() {
				var targetRefs []v1beta1.TargetRef

				// Deploy a Deployment and generate sample data inside Deployment
				deploy, err := f.DeployDeployment(fmt.Sprintf("source-deploy-%s", f.App()), int32(1))
				Expect(err).NotTo(HaveOccurred())
				_, err = f.GenerateSampleData(deploy.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())
				targetRefs = append(targetRefs, v1beta1.TargetRef{
					APIVersion: apps.SchemeGroupVersion.String(),
					Kind:       apis.KindDeployment,
					Name:       deploy.Name,
				})

				// Create new PVC and deploy a Pod to use this pvc
				// then generate sample data inside PVC
				pvc1, err := f.CreateNewPVC(fmt.Sprintf("source-pvc-%s", f.App()))
				Expect(err).NotTo(HaveOccurred())
				pod, err := f.DeployPod(pvc1.Name)
				Expect(err).NotTo(HaveOccurred())
				_, err = f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)
				Expect(err).NotTo(HaveOccurred())
				targetRefs = append(targetRefs, v1beta1.TargetRef{
					APIVersion: core.SchemeGroupVersion.String(),
					Kind:       apis.KindPersistentVolumeClaim,
					Name:       pvc1.Name,
				})

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupBatch, err := setupBackup(repo, func(in *v1beta1.BackupBatch) {
					for i, targetRef := range targetRefs {
						in.Spec.BackupConfigurationTemplates = append(in.Spec.BackupConfigurationTemplates, v1beta1.BackupConfigurationTemplate{
							Spec: v1beta1.BackupConfigurationTemplateSpec{
								Target: &v1beta1.BackupTarget{
									Ref: targetRef,
									Paths: []string{
										framework.TestSourceDataMountPath,
									},
									VolumeMounts: []core.VolumeMount{
										{
											Name:      framework.TestSourceDataVolumeName,
											MountPath: framework.TestSourceDataMountPath,
										},
									},
								},
							},
						})
						if targetRef.Kind == apis.KindPersistentVolumeClaim {
							in.Spec.BackupConfigurationTemplates[i].Spec.Task = v1beta1.TaskRef{
								Name: "pvc-backup",
							}
						} else {
							in.Spec.BackupConfigurationTemplates[i].Spec.Target.Paths = []string{
								framework.TestSourceDataMountPath,
							}
							in.Spec.BackupConfigurationTemplates[i].Spec.Target.VolumeMounts = []core.VolumeMount{
								{
									Name:      framework.TestSourceDataVolumeName,
									MountPath: framework.TestSourceDataMountPath,
								},
							}
						}
					}

				})
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupBatch.ObjectMeta, v1beta1.TargetRef{
					Name: backupBatch.Name,
					Kind: v1beta1.ResourceKindBackupBatch,
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

			})
		})
	})

})
