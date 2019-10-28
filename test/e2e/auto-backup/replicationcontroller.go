package auto_backup

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	store "kmodules.xyz/objectstore-api/api/v1"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
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
		deployReplicationController = func(name string) *core.ReplicationController {
			// Create PVC for ReplicationController
			pvc := f.CreateNewPVC(name)
			// Generate ReplicationController definition
			rc := f.ReplicationController(pvc.Name)
			rc.Name = name

			By(fmt.Sprintf("Creating ReplicationController: %s/%s ", rc.Namespace, rc.Name))
			createdRC, err := f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdRC)

			By("Waiting for ReplicationController to be ready")
			err = util.WaitUntilRCReady(f.KubeClient, createdRC.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdRC.ObjectMeta).Should(BeTrue())

			return createdRC
		}
	)

	Context("ReplicationController", func() {

		Context("Success Case", func() {

			It("should backup successfully", func() {
				// Create BackupBlueprint
				bb := f.CreateNewBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

				// Deploy a ReplicationController
				rc := deployReplicationController(fmt.Sprintf("rc-%s", f.App()))

				// Generate Sample Data
				f.GenerateSampleData(rc.ObjectMeta, apis.KindReplicationController)

				// create annotations for ReplicationController
				annotations := map[string]string{
					v1beta1.KeyBackupBlueprint: bb.Name,
					v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
					v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
				}
				// Adding and Ensuring annotations to Target
				f.AddAnnotationsToTarget(annotations, rc)

				// ensure Repository and BackupConfiguration
				backupConfig := f.CheckRepositoryAndBackupConfiguration(rc.ObjectMeta, apis.KindReplicationController)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)
			})
		})

		Context("Failure Case", func() {

			Context("Add inappropriate Repository and BackupConfiguration credential to BackupBlueprint", func() {
				It("should should fail BackupSession for missing Backend credential", func() {
					// Create Storage Secret for Minio
					secret := f.CreateBackendSecretForMinio()

					// Generate BackupBlueprint definition
					bb := f.BackupBlueprint(f.GetRepositoryInfo(secret.Name))
					bb.Spec.Backend.S3 = &store.S3Spec{}
					By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
					_, err := f.CreateBackupBlueprint(bb)
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(bb)

					// Deploy a ReplicationController
					rc := deployReplicationController(fmt.Sprintf("rc-%s", f.App()))

					// Generate Sample Data
					f.GenerateSampleData(rc.ObjectMeta, apis.KindReplicationController)

					// create annotations for ReplicationController
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, rc)

					// ensure Repository and BackupConfiguration
					backupConfig := f.CheckRepositoryAndBackupConfiguration(rc.ObjectMeta, apis.KindReplicationController)

					f.InstantBackupFailed(backupConfig.ObjectMeta)
				})
				It("should fail BackupSession for missing RetentionPolicy", func() {
					// Create Storage Secret for Minio
					secret := f.CreateBackendSecretForMinio()

					// Generate BackupBlueprint definition
					bb := f.BackupBlueprint(f.GetRepositoryInfo(secret.Name))
					bb.Spec.RetentionPolicy = v1alpha1.RetentionPolicy{}
					By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
					_, err := f.CreateBackupBlueprint(bb)
					Expect(err).NotTo(HaveOccurred())

					// Deploy a ReplicationController
					rc := deployReplicationController(fmt.Sprintf("rc-%s", f.App()))

					// Generate Sample Data
					f.GenerateSampleData(rc.ObjectMeta, apis.KindReplicationController)

					// create annotations for ReplicationController
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, rc)

					// ensure Repository and BackupConfiguration
					backupConfig := f.CheckRepositoryAndBackupConfiguration(rc.ObjectMeta, apis.KindReplicationController)

					// Take an Instant Backup the Sample Data
					f.InstantBackupFailed(backupConfig.ObjectMeta)
				})
			})

			Context("Add inappropriate annotation to Target", func() {
				It("should fail auto-backup for adding inappropriate BackupBlueprint annotation in ReplicationController", func() {
					// Create BackupBlueprint
					f.CreateNewBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

					// Deploy a ReplicationController
					rc := deployReplicationController(fmt.Sprintf("rc-%s", f.App()))

					// Generate Sample Data
					f.GenerateSampleData(rc.ObjectMeta, apis.KindReplicationController)

					// set wrong annotations to ReplicationController
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: framework.WrongBackupBlueprintName,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, rc)

					By("Will fail to get respective BackupBlueprint")
					getAnnotations := rc.GetAnnotations()
					_, err := f.GetBackupBlueprint(getAnnotations[v1beta1.KeyBackupBlueprint])
					Expect(err).To(HaveOccurred())
				})
				It("should fail BackupSession for adding inappropriate TargetPath/MountPath ReplicationController", func() {
					// Create BackupBlueprint
					bb := f.CreateNewBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

					// Deploy a ReplicationController
					rc := deployReplicationController(fmt.Sprintf("rc-%s", f.App()))

					// Generate Sample Data
					f.GenerateSampleData(rc.ObjectMeta, apis.KindReplicationController)

					// set wrong annotations to ReplicationController
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.WrongTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, rc)

					// ensure Repository and BackupConfiguration
					backupConfig := f.CheckRepositoryAndBackupConfiguration(rc.ObjectMeta, apis.KindReplicationController)

					// Take an Instant Backup the Sample Data
					f.InstantBackupFailed(backupConfig.ObjectMeta)
				})
			})

		})
	})

})
