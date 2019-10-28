package auto_backup

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	apps_util "kmodules.xyz/client-go/apps/v1"
	store "kmodules.xyz/objectstore-api/api/v1"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
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
		deployReplicaSet = func(name string) *apps.ReplicaSet {
			// Create PVC for ReplicaSet
			pvc := f.CreateNewPVC(name)
			// Generate ReplicaSet definition
			rs := f.ReplicaSet(pvc.Name)
			rs.Name = name

			By(fmt.Sprintf("Deploying ReplicaSet: %s/%s", rs.Namespace, rs.Name))
			createdRS, err := f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdRS)

			By("Waiting for ReplicaSet to be ready")
			err = apps_util.WaitUntilReplicaSetReady(f.KubeClient, createdRS.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdRS.ObjectMeta).Should(BeTrue())

			return createdRS
		}
	)

	Context("ReplicaSet", func() {

		Context("Success Case", func() {

			It("should success auto-backup for the ReplicaSet", func() {
				// Create BackupBlueprint
				bb := f.CreateNewBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

				// Deploy a ReplicaSet
				rs := deployReplicaSet(fmt.Sprintf("rs-%s", f.App()))

				// Generate Sample Data
				f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)

				// create annotations for ReplicaSet
				annotations := map[string]string{
					v1beta1.KeyBackupBlueprint: bb.Name,
					v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
					v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
				}
				// Adding and Ensuring annotations to Target
				f.AddAnnotationsToTarget(annotations, rs)

				// ensure Repository and BackupConfiguration
				backupConfig := f.CheckRepositoryAndBackupConfiguration(rs.ObjectMeta, apis.KindReplicaSet)

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

					// Deploy a ReplicaSet
					rs := deployReplicaSet(fmt.Sprintf("rs-%s", f.App()))

					// Generate Sample Data
					f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)

					// create annotations for ReplicaSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, rs)

					// ensure Repository and BackupConfiguration
					backupConfig := f.CheckRepositoryAndBackupConfiguration(rs.ObjectMeta, apis.KindReplicaSet)

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

					// Deploy a ReplicaSet
					rs := deployReplicaSet(fmt.Sprintf("rs-%s", f.App()))

					// Generate Sample Data
					f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)

					// create annotations for ReplicaSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, rs)

					// ensure Repository and BackupConfiguration
					backupConfig := f.CheckRepositoryAndBackupConfiguration(rs.ObjectMeta, apis.KindReplicaSet)

					f.InstantBackupFailed(backupConfig.ObjectMeta)
				})
			})

			Context("Add inappropriate annotation to Target", func() {
				It("should fail to get respective BackupBlueprint for adding wrong BackupBlueprint name", func() {
					// Create BackupBlueprint
					f.CreateNewBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))
					// Deploy a ReplicaSet
					rs := deployReplicaSet(fmt.Sprintf("rs-%s", f.App()))
					// Generate Sample Data
					f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)

					// set wrong annotations to ReplicaSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: framework.WrongBackupBlueprintName,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, rs)

					By("Will fail to get respective BackupBlueprint")
					getAnnotations := rs.GetAnnotations()
					_, err := f.GetBackupBlueprint(getAnnotations[v1beta1.KeyBackupBlueprint])
					Expect(err).To(HaveOccurred())
				})
				It("should fail BackupSession for adding inappropriate TargetPath/MountPath ReplicaSet", func() {
					// Create BackupBlueprint
					bb := f.CreateNewBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))
					// Deploy a ReplicaSet
					rs := deployReplicaSet(fmt.Sprintf("rs-%s", f.App()))
					// Generate Sample Data
					f.GenerateSampleData(rs.ObjectMeta, apis.KindReplicaSet)

					// set wrong annotations to ReplicaSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.WrongTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, rs)

					// ensure Repository and BackupConfiguration
					backupConfig := f.CheckRepositoryAndBackupConfiguration(rs.ObjectMeta, apis.KindReplicaSet)

					f.InstantBackupFailed(backupConfig.ObjectMeta)
				})

			})
		})
	})

})
