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
		deployDaemonSet = func(name string) *apps.DaemonSet {
			// Generate DaemonSet definition
			dmn := f.DaemonSet()
			dmn.Name = name

			By(fmt.Sprintf("Deploying DaemonSet: %s/%s", dmn.Namespace, dmn.Name))
			createdDmn, err := f.CreateDaemonSet(dmn)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdDmn)

			By("Waiting for DaemonSet to be ready")
			err = apps_util.WaitUntilDaemonSetReady(f.KubeClient, createdDmn.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdDmn.ObjectMeta).Should(BeTrue())

			return createdDmn
		}
	)

	Context("DaemonSet", func() {

		Context("Success Case", func() {

			It("should backup successfully", func() {
				// Create BackupBlueprint
				bb := f.CreateNewBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

				// Deploy a DaemonSet
				dmn := deployDaemonSet(fmt.Sprintf("dmn-%s", f.App()))

				// Generate Sample Data
				f.GenerateSampleData(dmn.ObjectMeta, apis.KindDaemonSet)

				// create proper annotations for Target
				annotations := map[string]string{
					v1beta1.KeyBackupBlueprint: bb.Name,
					v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
					v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
				}
				// Adding and Ensuring annotations to Target
				f.AddAnnotationsToTarget(annotations, dmn)

				// ensure Repository and BackupConfiguration
				backupConfig := f.CheckRepositoryAndBackupConfiguration(dmn.ObjectMeta, apis.KindDaemonSet)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)
			})
		})

		Context("Failure Case", func() {

			Context("Add inappropriate Repository and BackupConfiguration credential to BackupBlueprint", func() {
				It("should fail BackupSession for missing Backend credential", func() {
					// Create Storage Secret for Minio
					secret := f.CreateBackendSecretForMinio()

					// Generate BackupBlueprint definition
					bb := f.BackupBlueprint(f.GetRepositoryInfo(secret.Name))
					bb.Spec.Backend.S3 = &store.S3Spec{}
					By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
					_, err := f.CreateBackupBlueprint(bb)
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(bb)

					// Deploy a DaemonSet
					dmn := deployDaemonSet(fmt.Sprintf("dmn-%s", f.App()))

					// Generate Sample Data
					f.GenerateSampleData(dmn.ObjectMeta, apis.KindDaemonSet)

					// create annotations for DaemonSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, dmn)

					// ensure Repository and BackupConfiguration
					backupConfig := f.CheckRepositoryAndBackupConfiguration(dmn.ObjectMeta, apis.KindDaemonSet)

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

					// Deploy a DaemonSet
					dmn := deployDaemonSet(fmt.Sprintf("dmn-%s", f.App()))

					// Generate Sample Data
					f.GenerateSampleData(dmn.ObjectMeta, apis.KindDaemonSet)

					// create proper annotations for Target
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, dmn)

					// ensure Repository and BackupConfiguration
					backupConfig := f.CheckRepositoryAndBackupConfiguration(dmn.ObjectMeta, apis.KindDaemonSet)

					f.InstantBackupFailed(backupConfig.ObjectMeta)
				})
			})

			Context("Add inappropriate annotation to Target", func() {
				It("should fail to get respective BackupBlueprint for adding wrong BackupBlueprint name", func() {
					// Create BackupBlueprint
					f.CreateNewBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

					// Deploy a DaemonSet
					dmn := deployDaemonSet(fmt.Sprintf("dmn-%s", f.App()))

					// Generate Sample Data
					f.GenerateSampleData(dmn.ObjectMeta, apis.KindDaemonSet)

					// set wrong annotations to Deployment
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: framework.WrongBackupBlueprintName,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, dmn)

					By("Will fail to get respective BackupBlueprint")
					getAnnotations := dmn.GetAnnotations()
					_, err := f.GetBackupBlueprint(getAnnotations[v1beta1.KeyBackupBlueprint])
					Expect(err).To(HaveOccurred())
				})
				It("should fail BackupSession for adding inappropriate TargetPath/MountPath", func() {
					// Create BackupBlueprint
					bb := f.CreateNewBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

					// Deploy a DaemonSet
					dmn := deployDaemonSet(fmt.Sprintf("dmn-%s", f.App()))

					// Generate Sample Data
					f.GenerateSampleData(dmn.ObjectMeta, apis.KindDaemonSet)

					// set wrong annotations to DaemonSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.WrongTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, dmn)

					// ensure Repository and BackupConfiguration
					backupConfig := f.CheckRepositoryAndBackupConfiguration(dmn.ObjectMeta, apis.KindDaemonSet)

					f.InstantBackupFailed(backupConfig.ObjectMeta)
				})

			})
		})
	})

})
