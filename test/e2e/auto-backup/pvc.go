package auto_backup

import (
	"fmt"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"

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
			}
		}
	)

	Context("PVC", func() {

		Context("Success Case", func() {

			It("should backup successfully", func() {
				// Create BackupBlueprint
				bb := f.CreateBackupBlueprintForPVC(fmt.Sprintf("backupblueprint-%s", f.App()))

				// Create a PVC
				pvc := f.CreateNewPVC(fmt.Sprintf("pvc1-%s", f.App()))

				// Deploy a Pod
				pod := f.DeployPod(pvc.Name)

				// Generate Sample Data
				f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)

				// Add and Ensure annotations to Target
				f.AddAutoBackupAnnotations(annotations(bb.Name), pvc)

				// ensure Repository and BackupConfiguration
				backupConfig := f.VerifyAutoBackupConfigured(pvc.ObjectMeta, apis.KindPersistentVolumeClaim)

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
				It("should fail BackupSession for missing Backend credential", func() {
					// Create Secret for BackupBlueprint
					secret := f.CreateBackendSecretForMinio()

					// Generate BackupBlueprint definition
					bb := f.BackupBlueprint(secret.Name)
					bb.Spec.Backend.S3 = &store.S3Spec{}
					By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
					_, err := f.CreateBackupBlueprint(bb)
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(bb)

					// Create a PVC
					pvc := f.CreateNewPVC(fmt.Sprintf("pvc2-%s", f.App()))

					// Deploy a Pod
					pod := f.DeployPod(pvc.Name)

					// Generate Sample Data
					f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)

					// Add and Ensure annotations to Target
					f.AddAutoBackupAnnotations(annotations(bb.Name), pvc)

					// ensure Repository and BackupConfiguration
					backupConfig := f.VerifyAutoBackupConfigured(pvc.ObjectMeta, apis.KindPersistentVolumeClaim)

					// Take an Instant Backup the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has failed")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))
				})
				It("should fail BackupSession for missing RetentionPolicy", func() {
					// Create storage Secret for Minio
					secret := f.CreateBackendSecretForMinio()

					// Generate BackupBlueprint definition
					bb := f.BackupBlueprint(secret.Name)
					bb.Spec.RetentionPolicy = v1alpha1.RetentionPolicy{}
					By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
					_, err := f.CreateBackupBlueprint(bb)
					Expect(err).NotTo(HaveOccurred())

					// Create a PVC
					pvc := f.CreateNewPVC(fmt.Sprintf("pvc3-%s", f.App()))

					// Deploy a Pod
					pod := f.DeployPod(pvc.Name)

					// Generate Sample Data
					f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)

					// Add and Ensure annotations to Target
					f.AddAutoBackupAnnotations(annotations(bb.Name), pvc)

					// ensure Repository and BackupConfiguration
					backupConfig := f.VerifyAutoBackupConfigured(pvc.ObjectMeta, apis.KindPersistentVolumeClaim)

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
				It("should fail to create AutoBackup resources", func() {
					// Create BackupBlueprint
					bb := f.CreateBackupBlueprintForPVC(fmt.Sprintf("backupblueprint-%s", f.App()))

					// Create a PVC
					pvc := f.CreateNewPVC(fmt.Sprintf("pvc4-%s", f.App()))

					// Deploy a Pod
					pod := f.DeployPod(pvc.Name)

					// Generate Sample Data
					f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)

					anno := annotations(bb.Name)
					anno[v1beta1.KeyBackupBlueprint] = framework.WrongBackupBlueprintName
					f.AddAutoBackupAnnotations(anno, pvc)

					By("Will fail to get respective BackupBlueprint")
					getAnnotations := pvc.GetAnnotations()
					_, err := f.GetBackupBlueprint(getAnnotations[v1beta1.KeyBackupBlueprint])
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})

})
