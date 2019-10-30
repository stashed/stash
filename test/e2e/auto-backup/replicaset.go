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

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	var (
		annotations = func(backupBlueprintName string) map[string]string {
			return map[string]string{
				v1beta1.KeyBackupBlueprint: backupBlueprintName,
				v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
				v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
			}
		}
	)

	Context("ReplicaSet", func() {

		Context("Success Case", func() {

			It("should success auto-backup for the ReplicaSet", func() {
				// Create BackupBlueprint
				bb := f.CreateBackupBlueprintForWorkload(fmt.Sprintf("backupblueprint-%s", f.App()))

				// Deploy a ReplicaSet
				rs := f.DeployReplicaSet(fmt.Sprintf("rs1-%s", f.App()), int32(1))

				// Generate Sample Data
				f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)

				// Add and Ensure annotations to Target
				f.AddAutoBackupAnnotations(annotations(bb.Name), rs)

				// ensure Repository and BackupConfiguration
				backupConfig := f.VerifyAutoBackupConfigured(rs.ObjectMeta, apis.KindReplicaSet)

				// Take an Instant Backup the Sample Data
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
				It("should should fail BackupSession for missing Backend credential", func() {
					// Create Secret for BackupBlueprint
					secret := f.CreateBackendSecretForMinio()

					// Generate BackupBlueprint definition
					bb := f.BackupBlueprint(secret.Name)
					bb.Spec.Backend.S3 = &store.S3Spec{}
					By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
					_, err := f.CreateBackupBlueprint(bb)
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(bb)

					// Deploy a ReplicaSet
					rs := f.DeployReplicaSet(fmt.Sprintf("rs2-%s", f.App()), int32(1))

					// Generate Sample Data
					f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)

					// Add and Ensure annotations to Target
					f.AddAutoBackupAnnotations(annotations(bb.Name), rs)

					// ensure Repository and BackupConfiguration
					backupConfig := f.VerifyAutoBackupConfigured(rs.ObjectMeta, apis.KindReplicaSet)

					// Take an Instant Backup the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has failed")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))
				})
				It("should fail BackupSession for missing RetentionPolicy", func() {
					// Create Storage Secret for Minio
					secret := f.CreateBackendSecretForMinio()

					// Generate BackupBlueprint definition
					bb := f.BackupBlueprint(secret.Name)
					bb.Spec.RetentionPolicy = v1alpha1.RetentionPolicy{}
					By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
					_, err := f.CreateBackupBlueprint(bb)
					Expect(err).NotTo(HaveOccurred())

					// Deploy a ReplicaSet
					rs := f.DeployReplicaSet(fmt.Sprintf("rs3-%s", f.App()), int32(1))

					// Generate Sample Data
					f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)

					// Add and Ensure annotations to Target
					f.AddAutoBackupAnnotations(annotations(bb.Name), rs)

					// ensure Repository and BackupConfiguration
					backupConfig := f.VerifyAutoBackupConfigured(rs.ObjectMeta, apis.KindReplicaSet)

					// Take an Instant Backup the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has failed")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))
				})
			})

			Context("Add inappropriate annotation to Target", func() {
				FIt("should fail to create AutoBackup resources", func() {
					// Create BackupBlueprint
					f.CreateBackupBlueprintForWorkload(fmt.Sprintf("backupblueprint-%s", f.App()))

					// Deploy a ReplicaSet
					rs := f.DeployReplicaSet(fmt.Sprintf("rs4-%s", f.App()), int32(1))

					// Generate Sample Data
					f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)

					// Add and Ensure annotations to Target
					f.AddAutoBackupAnnotations(annotations(framework.WrongBackupBlueprintName), rs)

					f.EventuallyAutoBackup(rs.ObjectMeta, apis.KindReplicaSet).Should(matcher.HaveEvent(eventer.EventReasonAutoBackupResourcesCreationFailed))
				})
				It("should fail BackupSession for adding inappropriate TargetPath/MountPath", func() {
					// Create BackupBlueprint
					bb := f.CreateBackupBlueprintForWorkload(fmt.Sprintf("backupblueprint-%s", f.App()))

					// Deploy a ReplicaSet
					rs := f.DeployReplicaSet(fmt.Sprintf("rs5-%s", f.App()), int32(1))

					// Generate Sample Data
					f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)

					// Add and Ensure annotations to Target
					anno := annotations(bb.Name)
					anno[v1beta1.KeyTargetPaths] = framework.WrongTargetPath
					f.AddAutoBackupAnnotations(anno, rs)

					// ensure Repository and BackupConfiguration
					backupConfig := f.VerifyAutoBackupConfigured(rs.ObjectMeta, apis.KindReplicaSet)

					// Take an Instant Backup the Sample Data
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
