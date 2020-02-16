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
package misc

import (
	"fmt"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/test/e2e/framework"
	"stash.appscode.dev/stash/test/e2e/matcher"

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	store "kmodules.xyz/objectstore-api/api/v1"
)

var _ = Describe("Repository", func() {

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

	Context("Local Backend", func() {
		Context("Invalid MountPath", func() {
			It("should reject to create Repository for using `/stash` as `mountPath`", func() {
				// Deploy a Deployment
				_, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				// Create Storage Secret
				By("Creating Storage Secret")
				cred := f.SecretForLocalBackend()
				_, err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(&cred)

				// We are going to use an PVC as backend
				pvc, err := f.CreateNewPVC(rand.WithUniqSuffix("backend-pvc"))
				Expect(err).NotTo(HaveOccurred())

				// Generate Repository Definition
				repo := f.NewLocalRepositoryWithPVC(cred.Name, pvc.Name)

				// Use `/stash` as `mountPath`
				repo.Spec.Backend.Local.MountPath = "/stash"

				// reject to create Repository
				By("reject to create Repository")
				_, err = f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(repo)
				Expect(err).To(HaveOccurred())
			})

			It("should reject to create Repository for using `/stash/data` as `mountPath`", func() {
				// Deploy a Deployment
				_, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				// Create Storage Secret
				By("Creating Storage Secret")
				cred := f.SecretForLocalBackend()
				_, err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(&cred)

				// We are going to use an PVC as backend
				pvc, err := f.CreateNewPVC(rand.WithUniqSuffix("backend-pvc"))
				Expect(err).NotTo(HaveOccurred())

				// Generate Repository Definition
				repo := f.NewLocalRepositoryWithPVC(cred.Name, pvc.Name)

				// Use `/stash` as `mountPath`
				repo.Spec.Backend.Local.MountPath = "/stash/data"

				// reject to create Repository
				By("reject to create Repository")
				_, err = f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Create(repo)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("Invalid MountPath in Auto-Backup", func() {
			annotations := func(backupBlueprintName string) map[string]string {
				return map[string]string{
					v1beta1.KeyBackupBlueprint: backupBlueprintName,
					v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
					v1beta1.KeyVolumeMounts:    framework.TestSourceVolumeAndMount,
				}
			}

			It("should reject to create auto-backup resources for using `/stash` as `mountPath`", func() {
				// Create Secret for BackupBlueprint
				cred := f.SecretForLocalBackend()
				_, err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(&cred)

				// We are going to use an PVC as backend
				pvc, err := f.CreateNewPVC(rand.WithUniqSuffix("backend-pvc"))
				Expect(err).NotTo(HaveOccurred())

				// Generate BackupBlueprint definition
				bb := f.BackupBlueprint(cred.Name)
				bb.Spec.Backend.Local = &store.LocalSpec{
					VolumeSource: core.VolumeSource{
						PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc.Name,
						},
					},
					MountPath: "/stash", // Use `/stash` as `mountPath`, same thing happened if you use `/stash/data` as `mountPath`
				}

				By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
				createdBB, err := f.CreateBackupBlueprint(bb)
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(createdBB)

				// Deploy a DaemonSet
				deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				// Add auto-backup annotations to Target
				err = f.AddAutoBackupAnnotations(annotations(bb.Name), deployment)
				Expect(err).NotTo(HaveOccurred())

				// AutoBackup Resource creation failed
				f.EventuallyEvent(deployment.ObjectMeta, apis.KindDeployment).Should(matcher.HaveEvent(eventer.EventReasonAutoBackupResourcesCreationFailed))
			})
		})
	})
})
