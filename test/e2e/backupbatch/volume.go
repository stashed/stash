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
	"stash.appscode.dev/stash/test/e2e/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Volumes", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	var (
		setupPVCBackup = func(repo *api.Repository, transformFunc func(in *v1beta1.BackupBatch)) (*v1beta1.BackupBatch, error) {
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

			return createdBackupBatch, err
		}
	)

	Context("Backup", func() {

		Context("Backup Volumes using BackupBatch", func() {
			It("should Backup Volumes", func() {
				var targetRefs []v1beta1.TargetRef

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

				// Create another new PVC and deploy a Pod to use this pvc
				// then generate sample data inside PVC
				pvc2, err := f.CreateNewPVC(fmt.Sprintf("config-pvc-%s", f.App()))
				Expect(err).NotTo(HaveOccurred())
				pod, err = f.DeployPod(pvc2.Name)
				Expect(err).NotTo(HaveOccurred())
				_, err = f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)
				Expect(err).NotTo(HaveOccurred())
				targetRefs = append(targetRefs, v1beta1.TargetRef{
					APIVersion: core.SchemeGroupVersion.String(),
					Kind:       apis.KindPersistentVolumeClaim,
					Name:       pvc2.Name,
				})

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Volume Backup
				backupBatch, err := setupPVCBackup(repo, func(in *v1beta1.BackupBatch) {
					for _, targetRef := range targetRefs {
						in.Spec.BackupConfigurationTemplates = append(in.Spec.BackupConfigurationTemplates, v1beta1.BackupConfigurationTemplate{
							Spec: v1beta1.BackupConfigurationTemplateSpec{
								Task: v1beta1.TaskRef{
									Name: "pvc-backup",
								},
								Target: &v1beta1.BackupTarget{
									Ref: targetRef,
								},
							},
						})
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
