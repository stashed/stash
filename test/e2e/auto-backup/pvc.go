package auto_backup

import (
	"fmt"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	v1 "kmodules.xyz/client-go/core/v1"
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
		deployPod = func(pvcName string) *core.Pod {
			// Generate Pod definition
			pod := f.Pod(pvcName)

			By(fmt.Sprintf("Deploying Pod: %s/%s", pod.Namespace, pod.Name))
			createdPod, err := f.CreatePod(pod)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdPod)

			By("Waiting for Pod to be ready")
			err = v1.WaitUntilPodRunning(f.KubeClient, createdPod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdPod.ObjectMeta).Should(BeTrue())

			return createdPod
		}
	)

	Context("PVC", func() {

		Context("Success Case", func() {

			It("should backup successfully", func() {
				// Create BackupBlueprint
				bb := f.CreateNewBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

				// Create a PVC
				pvc := f.CreateNewPVC(fmt.Sprintf("pvc-%s", f.App()))

				// Deploy a Pod
				pod := deployPod(pvc.Name)

				// Generate Sample Data
				f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)

				// Create annotation for Target
				annotations := map[string]string{
					v1beta1.KeyBackupBlueprint: bb.Name,
					v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
					v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
				}
				// Adding and Ensuring annotations to Target
				f.AddAnnotationsToTarget(annotations, pvc)

				// ensure Repository and BackupConfiguration
				backupConfig := f.CheckRepositoryAndBackupConfiguration(pvc.ObjectMeta, apis.KindPersistentVolumeClaim)

				// Take an Instant Backup the Sample Data
				f.TakeInstantBackup(backupConfig.ObjectMeta)
			})
		})

		Context("Failure Case", func() {

			Context("Add inappropriate Repository and BackupConfiguration credential to BackupBlueprint", func() {
				It("should fail BackupSession for missing Backend credential", func() {
					// Create storage Secret for Minio
					secret := f.CreateBackendSecretForMinio()

					// Generate BackupBlueprint definition
					bb := f.BackupBlueprint(f.GetRepositoryInfo(secret.Name))
					bb.Spec.Backend.S3 = &store.S3Spec{}
					By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
					_, err := f.CreateBackupBlueprint(bb)
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(bb)

					// Create a PVC
					pvc := f.CreateNewPVC(fmt.Sprintf("pvc-%s", f.App()))

					// Deploy a Pod
					pod := deployPod(pvc.Name)

					// Generate Sample Data
					f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)

					// create annotations for Deployment
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, pvc)

					// ensure Repository and BackupConfiguration
					backupConfig := f.CheckRepositoryAndBackupConfiguration(pvc.ObjectMeta, apis.KindPersistentVolumeClaim)

					f.InstantBackupFailed(backupConfig.ObjectMeta)
				})
				It("should fail BackupSession for missing RetentionPolicy", func() {
					// Create storage Secret for Minio
					secret := f.CreateBackendSecretForMinio()

					// Generate BackupBlueprint definition
					bb := f.BackupBlueprint(f.GetRepositoryInfo(secret.Name))
					bb.Spec.RetentionPolicy = v1alpha1.RetentionPolicy{}
					By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
					_, err := f.CreateBackupBlueprint(bb)
					Expect(err).NotTo(HaveOccurred())

					// Create a PVC
					pvc := f.CreateNewPVC(fmt.Sprintf("pvc-%s", f.App()))

					// Deploy a Pod
					pod := deployPod(pvc.Name)

					// Generate Sample Data
					f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)

					// create annotations for Deployment
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, pvc)

					// ensure Repository and BackupConfiguration
					backupConfig := f.CheckRepositoryAndBackupConfiguration(pvc.ObjectMeta, apis.KindPersistentVolumeClaim)

					f.InstantBackupFailed(backupConfig.ObjectMeta)
				})
			})

			Context("Add inappropriate annotation to Target", func() {
				It("should fail to get respective BackupBlueprint for adding wrong BackupBlueprint name", func() {
					// Create BackupBlueprint
					f.CreateNewBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

					// Create a PVC
					pvc := f.CreateNewPVC(fmt.Sprintf("pvc-%s", f.App()))

					// Deploy a Pod
					pod := deployPod(pvc.Name)

					// Generate Sample Data
					f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)

					// set wrong annotations to Deployment
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: framework.WrongBackupBlueprintName,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					f.AddAnnotationsToTarget(annotations, pvc)

					By("Will fail to get respective BackupBlueprint")
					getAnnotations := pvc.GetAnnotations()
					_, err := f.GetBackupBlueprint(getAnnotations[v1beta1.KeyBackupBlueprint])
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})

})
