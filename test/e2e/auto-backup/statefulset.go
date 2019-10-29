package auto_backup

import (
	"fmt"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	apps_util "kmodules.xyz/client-go/apps/v1"
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
		deploySS = func(name string) *apps.StatefulSet {
			// Generate StatefulSet definition
			ss := f.StatefulSetForV1beta1API()
			ss.Name = name

			By(fmt.Sprintf("Deploying StatefulSet: %s/%s", ss.Namespace, ss.Name))
			createdSS, err := f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdSS)

			By("Waiting for StatefulSet to be ready")
			err = apps_util.WaitUntilStatefulSetReady(f.KubeClient, createdSS.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdSS.ObjectMeta).Should(BeTrue())

			return createdSS
		}
	)

	Context("StatefulSet", func() {

		Context("Success Case", func() {

			It("should success auto-backup for the StatefulSet", func() {
				// Create BackupBlueprint
				bb := f.CreateNewBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

				// Deploy a StatefulSet
				ss := deploySS(fmt.Sprintf("ss-%s", f.App()))

				// Generate Sample Data
				f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)

				// create annotations for StatefulSet
				annotations := map[string]string{
					v1beta1.KeyBackupBlueprint: bb.Name,
					v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
					v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
				}
				// Adding and Ensuring annotations to Target
				f.AddAnnotationsToTarget(annotations, ss)

				// ensure Repository and BackupConfiguration
				backupConfig := f.CheckRepositoryAndBackupConfiguration(ss.ObjectMeta, apis.KindStatefulSet)

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

					// Deploy a StatefulSet
					ss := deploySS(fmt.Sprintf("ss-%s", f.App()))

					// Generate Sample Data
					f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)

					// create annotations for StatefulSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, ss)

					// ensure Repository and BackupConfiguration
					backupConfig := f.CheckRepositoryAndBackupConfiguration(ss.ObjectMeta, apis.KindStatefulSet)

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

					// Deploy a StatefulSet
					ss := deploySS(fmt.Sprintf("ss-%s", f.App()))

					// Generate Sample Data
					f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)

					// create annotations for StatefulSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, ss)

					// ensure Repository and BackupConfiguration
					backupConfig := f.CheckRepositoryAndBackupConfiguration(ss.ObjectMeta, apis.KindStatefulSet)

					// Take an Instant Backup the Sample Data
					f.InstantBackupFailed(backupConfig.ObjectMeta)
				})
			})

			Context("Add inappropriate annotation to Target", func() {
				It("Should fail auto-backup for adding inappropriate BackupBlueprint annotation in StatefulSet", func() {
					// Create BackupBlueprint
					f.CreateNewBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

					// Deploy a StatefulSet
					ss := deploySS(fmt.Sprintf("ss-%s", f.App()))

					// Generate Sample Data
					f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)

					// set wrong annotations to StatefulSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: framework.WrongBackupBlueprintName,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, ss)

					By("Will fail to get respective BackupBlueprint")
					getAnnotations := ss.GetAnnotations()
					_, err := f.GetBackupBlueprint(getAnnotations[v1beta1.KeyBackupBlueprint])
					Expect(err).To(HaveOccurred())
				})
				It("should fail BackupSession for adding inappropriate TargetPath/MountPath StatefulSet", func() {
					// Create BackupBlueprint
					bb := f.CreateNewBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

					// Deploy a StatefulSet
					ss := deploySS(fmt.Sprintf("ss-%s", f.App()))

					// Generate Sample Data
					f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)

					// set wrong annotations to StatefulSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.WrongTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, ss)

					// ensure Repository and BackupConfiguration
					backupConfig := f.CheckRepositoryAndBackupConfiguration(ss.ObjectMeta, apis.KindStatefulSet)

					// Take an Instant Backup the Sample Data
					f.InstantBackupFailed(backupConfig.ObjectMeta)
				})
			})
		})
	})

})
