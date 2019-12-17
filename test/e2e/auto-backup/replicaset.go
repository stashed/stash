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

package auto_backup

import (
	"fmt"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/test/e2e/framework"
	"stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	store "kmodules.xyz/objectstore-api/api/v1"
)

var _ = Describe("Auto-Backup", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			f.PrintDebugHelpers()
		}
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	annotations := func(backupBlueprintName string) map[string]string {
		return map[string]string{
			v1beta1.KeyBackupBlueprint: backupBlueprintName,
			v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
			v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
		}
	}

	Context("ReplicaSet", func() {

		Context("Success Case", func() {

			It("should success auto-backup for the ReplicaSet", func() {
				// Create BackupBlueprint
				bb, err := f.CreateBackupBlueprintForWorkload(fmt.Sprintf("backupblueprint-%s", f.App()))
				Expect(err).NotTo(HaveOccurred())

				// Deploy a ReplicaSet
				rs, err := f.DeployReplicaSet(fmt.Sprintf("rs1-%s", f.App()), int32(1))
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				_, err = f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)
				Expect(err).NotTo(HaveOccurred())

				// Add and Ensure annotations to Target
				err = f.AddAutoBackupAnnotations(annotations(bb.Name), rs)
				Expect(err).NotTo(HaveOccurred())

				// ensure Repository and BackupConfiguration
				backupConfig, err := f.VerifyAutoBackupConfigured(rs.ObjectMeta, apis.KindReplicaSet)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup of the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
			})
		})

		Context("Failure Case", func() {

			Context("Missing AutoBackup resource credential in BackupBlueprint", func() {
				It("should fail BackupSession for missing Backend credential", func() {
					// Create Secret for BackupBlueprint
					secret, err := f.CreateBackendSecretForMinio()
					Expect(err).NotTo(HaveOccurred())

					// Generate BackupBlueprint definition
					bb := f.BackupBlueprint(secret.Name)
					bb.Spec.Backend.S3 = &store.S3Spec{}
					By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
					_, err = f.CreateBackupBlueprint(bb)
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(bb)

					// Deploy a ReplicaSet
					rs, err := f.DeployReplicaSet(fmt.Sprintf("rs2-%s", f.App()), int32(1))
					Expect(err).NotTo(HaveOccurred())

					// Generate Sample Data
					_, err = f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)
					Expect(err).NotTo(HaveOccurred())

					// Add and Ensure annotations to Target
					err = f.AddAutoBackupAnnotations(annotations(bb.Name), rs)
					Expect(err).NotTo(HaveOccurred())

					// ensure Repository and BackupConfiguration
					backupConfig, err := f.VerifyAutoBackupConfigured(rs.ObjectMeta, apis.KindReplicaSet)
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup of the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has failed")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))
				})
				It("should fail BackupSession for missing RetentionPolicy", func() {
					// Create Storage Secret for Minio
					secret, err := f.CreateBackendSecretForMinio()
					Expect(err).NotTo(HaveOccurred())

					// Generate BackupBlueprint definition
					bb := f.BackupBlueprint(secret.Name)
					bb.Spec.RetentionPolicy = v1alpha1.RetentionPolicy{}
					By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
					_, err = f.CreateBackupBlueprint(bb)
					Expect(err).NotTo(HaveOccurred())

					// Deploy a ReplicaSet
					rs, err := f.DeployReplicaSet(fmt.Sprintf("rs3-%s", f.App()), int32(1))
					Expect(err).NotTo(HaveOccurred())

					// Generate Sample Data
					_, err = f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)
					Expect(err).NotTo(HaveOccurred())

					// Add and Ensure annotations to Target
					err = f.AddAutoBackupAnnotations(annotations(bb.Name), rs)
					Expect(err).NotTo(HaveOccurred())

					// ensure Repository and BackupConfiguration
					backupConfig, err := f.VerifyAutoBackupConfigured(rs.ObjectMeta, apis.KindReplicaSet)
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup of the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has failed")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))
				})
			})

			Context("Add inappropriate annotation to Target", func() {
				It("should fail to create AutoBackup resources", func() {
					// Create BackupBlueprint
					_, err := f.CreateBackupBlueprintForWorkload(fmt.Sprintf("backupblueprint-%s", f.App()))
					Expect(err).NotTo(HaveOccurred())

					// Deploy a ReplicaSet
					rs, err := f.DeployReplicaSet(fmt.Sprintf("rs4-%s", f.App()), int32(1))
					Expect(err).NotTo(HaveOccurred())

					// Generate Sample Data
					_, err = f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)
					Expect(err).NotTo(HaveOccurred())

					// Add and Ensure annotations to Target
					err = f.AddAutoBackupAnnotations(annotations(framework.WrongBackupBlueprintName), rs)
					Expect(err).NotTo(HaveOccurred())

					// AutoBackup Resource creation failed
					f.EventuallyEvent(rs.ObjectMeta, apis.KindReplicaSet).Should(matcher.HaveEvent(eventer.EventReasonAutoBackupResourcesCreationFailed))
				})
				It("should fail BackupSession for adding inappropriate TargetPath/MountPath", func() {
					// Create BackupBlueprint
					bb, err := f.CreateBackupBlueprintForWorkload(fmt.Sprintf("backupblueprint-%s", f.App()))
					Expect(err).NotTo(HaveOccurred())

					// Deploy a ReplicaSet
					rs, err := f.DeployReplicaSet(fmt.Sprintf("rs5-%s", f.App()), int32(1))
					Expect(err).NotTo(HaveOccurred())

					// Generate Sample Data
					_, err = f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)
					Expect(err).NotTo(HaveOccurred())

					// Add and Ensure annotations to Target
					anno := annotations(bb.Name)
					anno[v1beta1.KeyTargetPaths] = framework.WrongTargetPath
					err = f.AddAutoBackupAnnotations(anno, rs)
					Expect(err).NotTo(HaveOccurred())

					// ensure Repository and BackupConfiguration
					backupConfig, err := f.VerifyAutoBackupConfigured(rs.ObjectMeta, apis.KindReplicaSet)
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup of the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has failed")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))
				})

			})
		})
	})

})
