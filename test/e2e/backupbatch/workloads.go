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
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Workloads", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Backup", func() {

		Context("Backup Workloads using BackupBatch", func() {
			It("should Backup Workloads", func() {
				var targetRefs []v1beta1.TargetRef

				// Deploy a Deployment and generate sample data inside Deployment
				deploy, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())
				_, err = f.GenerateSampleData(deploy.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())
				targetRefs = append(targetRefs, v1beta1.TargetRef{
					APIVersion: apps.SchemeGroupVersion.String(),
					Kind:       apis.KindDeployment,
					Name:       deploy.Name,
				})

				// Deploy a DaemonSet and generate sample data inside DaemonSet
				dmn, err := f.DeployDaemonSet(framework.SourceDaemonSet, framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())
				_, err = f.GenerateSampleData(dmn.ObjectMeta, apis.KindDaemonSet)
				Expect(err).NotTo(HaveOccurred())
				targetRefs = append(targetRefs, v1beta1.TargetRef{
					APIVersion: apps.SchemeGroupVersion.String(),
					Kind:       apis.KindDaemonSet,
					Name:       dmn.Name,
				})

				// Deploy a StatefulSet and generate sample data inside StatefulSet
				sts, err := f.DeployStatefulSet(framework.SourceStatefulSet, int32(3), framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())
				_, err = f.GenerateSampleData(sts.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())
				targetRefs = append(targetRefs, v1beta1.TargetRef{
					APIVersion: apps.SchemeGroupVersion.String(),
					Kind:       apis.KindStatefulSet,
					Name:       sts.Name,
				})

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupBatch, err := f.SetupWorkloadBackupForBackupBatch(targetRefs, repo, func(in *v1beta1.BackupBatch) {
					for _, targetRef := range targetRefs {
						in.Spec.Members = append(in.Spec.Members, v1beta1.BackupConfigurationTemplateSpec{
							Target: &v1beta1.BackupTarget{
								Ref: targetRef,
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
