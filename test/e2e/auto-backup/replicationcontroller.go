package auto_backup

import (
	"fmt"
	"strings"

	"github.com/appscode/go/sets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	store "kmodules.xyz/objectstore-api/api/v1"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"
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
		createBackendSecretForMinio = func() *core.Secret {
			// Create Storage Secret
			By("Creating Storage Secret for Minio")
			cred := f.SecretForMinioBackend(true)

			if missing, _ := BeZero().Match(cred); missing {
				Skip("Missing Minio credential")
			}
			createdCred, err := f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(&cred)

			return createdCred
		}

		getRepositoryInfo = func(secretName string) v1alpha1.RepositorySpec {
			repoInfo := v1alpha1.RepositorySpec{
				Backend: store.Backend{
					S3: &store.S3Spec{
						Endpoint: f.MinioServiceAddres(),
						Bucket:   "minio-bucket",
						Prefix:   fmt.Sprintf("stash-e2e/%s/%s", f.Namespace(), f.App()),
					},
					StorageSecretName: secretName,
				},
				WipeOut: false,
			}
			return repoInfo
		}

		createBackupBlueprint = func(name string) *v1beta1.BackupBlueprint {
			// Create Secret for BackupBlueprint
			secret := createBackendSecretForMinio()

			// Generate BackupBlueprint definition
			bb := f.BackupBlueprint(getRepositoryInfo(secret.Name))
			bb.Name = name

			By(fmt.Sprintf("Creating BackupBlueprint: %s ", bb.Name))
			createdBB, err := f.CreateBackupBlueprint(bb)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdBB)
			return createdBB
		}

		createPVC = func(name string) *core.PersistentVolumeClaim {
			// Generate PVC definition
			pvc := f.PersistentVolumeClaim()
			pvc.Name = fmt.Sprintf("%s-pvc-%s", strings.Split(name, "-")[0], f.App())

			By("Creating PVC: " + pvc.Name)
			createdPVC, err := f.CreatePersistentVolumeClaim(pvc)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdPVC)

			return createdPVC
		}

		deployReplicationController = func(name string) *core.ReplicationController {
			// Create PVC for ReplicationController
			pvc := createPVC(name)
			// Generate ReplicationController definition
			rc := f.ReplicationController(pvc.Name)
			rc.Name = name

			By("Deploying ReplicationController: " + rc.Name)
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

		generateSampleData = func(rc *core.ReplicationController) sets.String {
			By("Generating sample data inside ReplicationController pods")
			err := f.CreateSampleDataInsideWorkload(rc.ObjectMeta, apis.KindReplicationController)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that sample data has been generated")
			sampleData, err := f.ReadSampleDataFromFromWorkload(rc.ObjectMeta, apis.KindReplicationController)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleData).ShouldNot(BeEmpty())

			return sampleData
		}

		addAnnotationsToWorkload = func(annotations map[string]string, rc *core.ReplicationController) {
			By(fmt.Sprintf("Adding auto-backup specific annotations to the ReplicationController: %s/%s", rc.Namespace, rc.Name))
			err := f.AddAutoBackupAnnotationsToTarget(annotations, rc)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the auto-backup annotations has been added successfully")
			f.EventuallyAutoBackupAnnotationsFound(annotations, rc).Should(BeTrue())
		}

		checkBackupSessionSucceeded = func(rc *core.ReplicationController) {
			// In BackupBlueprint Stash operator create BackupConfiguration and Repository
			// such that the name of the BackupConfiguration and Repository will follow
			// the patter: <lower case of the workload kind>-<workload name>.
			// we will form the ObjectMeta name and namespace for farther process.
			objMeta := metav1.ObjectMeta{
				Name:      fmt.Sprintf("replicationcontroller-%s", rc.Name),
				Namespace: f.Namespace(),
			}
			By("Waiting for Repository")
			f.EventuallyRepositoryCreated(objMeta).Should(BeTrue())

			By("Waiting for BackupConfiguration")
			f.EventuallyBackupConfigurationCreated(objMeta).Should(BeTrue())

			By("Verifying that backup triggering CronJob has been created")
			f.EventuallyCronJobCreated(objMeta).Should(BeTrue())

			By("Verifying that sidecar has been injected")
			f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for ReplicationController to be ready with sidecar")
			err := f.WaitUntilRCReadyWithSidecar(rc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for BackupSession")
			f.EventuallyBackupSessionCreated(objMeta).Should(BeTrue())

			By("Waiting for backup process to complete")
			bs, err := f.GetBackupSession(objMeta)
			Expect(err).NotTo(HaveOccurred())
			f.EventuallyBackupProcessCompleted(bs.ObjectMeta).Should(BeTrue())

			By("Verifying that BackupSession has succeeded")
			completedBS, err := f.StashClient.StashV1beta1().BackupSessions(bs.Namespace).Get(bs.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
		}

		checkBackupSessionFailed = func(rc *core.ReplicationController) {
			// BackupBlueprint create BackupConfiguration and Repository such that
			// the name of the BackupConfiguration and Repository will follow
			// the patter: <lower case of the workload kind>-<workload name>.
			// we will form the meta name and namespace using this pattern for farther process.
			meta := metav1.ObjectMeta{
				Name:      fmt.Sprintf("replicationcontroller-%s", rc.Name),
				Namespace: f.Namespace(),
			}
			By("Waiting for Repository")
			f.EventuallyRepositoryCreated(meta).Should(BeTrue())

			By("Waiting for BackupConfiguration")
			f.EventuallyBackupConfigurationCreated(meta).Should(BeTrue())

			By("Waiting for BackupSession")
			f.EventuallyBackupSessionCreated(meta).Should(BeTrue())

			By("Waiting for backup process to complete")
			bs, err := f.GetBackupSession(meta)
			Expect(err).NotTo(HaveOccurred())
			f.EventuallyBackupProcessCompleted(bs.ObjectMeta).Should(BeTrue())

			By("Verifying that BackupSession has failed")
			completedBS, err := f.StashClient.StashV1beta1().BackupSessions(bs.Namespace).Get(bs.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))
		}
	)

	Context("ReplicationController", func() {

		Context("Success event: ", func() {

			It("should success auto-backup for the ReplicationController", func() {
				// Create BackupBlueprint
				bb := createBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))
				// Deploy a ReplicationController
				rc := deployReplicationController(fmt.Sprintf("rc-%s", f.App()))
				// Generate Sample Data
				generateSampleData(rc)
				// create proper annotations for ReplicationController auto-backup
				annotations := map[string]string{
					v1beta1.KeyBackupBlueprint: bb.Name,
					v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
					v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
				}
				// Adding and Ensuring annotations to workload
				addAnnotationsToWorkload(annotations, rc)
				// check Stash Object status
				checkBackupSessionSucceeded(rc)
			})
		})

		Context("Failure event: ", func() {
			It("Should fail for adding inappropriate Repository secret in BackupBlueprint", func() {
				// Create Storage Secret for Minio
				secret := createBackendSecretForMinio()

				// Generate BackupBlueprint definition
				bb := f.BackupBlueprint(getRepositoryInfo(secret.Name))
				bb.Spec.Backend.StorageSecretName = ""
				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", bb.Name))
				_, err := f.CreateBackupBlueprint(bb)
				Expect(err).To(HaveOccurred())
				f.AppendToCleanupList(bb)
			})
			It("Should fail for adding inappropriate Repository backend in BackupBlueprint", func() {
				// Create Storage Secret for Minio
				secret := createBackendSecretForMinio()

				// Generate BackupBlueprint definition
				bb := f.BackupBlueprint(getRepositoryInfo(secret.Name))
				bb.Spec.Backend.S3 = &store.S3Spec{}
				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", bb.Name))
				_, err := f.CreateBackupBlueprint(bb)
				Expect(err).To(HaveOccurred())
				f.AppendToCleanupList(bb)
				// Deploy a ReplicationController
				rc := deployReplicationController(fmt.Sprintf("rc-%s", f.App()))
				// Generate Sample Data
				generateSampleData(rc)
				// create proper annotations for ReplicationController auto-backup
				annotations := map[string]string{
					v1beta1.KeyBackupBlueprint: bb.Name,
					v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
					v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
				}
				// Adding and Ensuring annotations to workload
				addAnnotationsToWorkload(annotations, rc)
				// check Stash Object status
				checkBackupSessionFailed(rc)
			})
			It("Should fail for adding inappropriate BackupConfiguration RetentionPolicy in BackupBlueprint", func() {
				// Create Storage Secret for Minio
				secret := createBackendSecretForMinio()

				// Generate BackupBlueprint definition
				bb := f.BackupBlueprint(getRepositoryInfo(secret.Name))
				bb.Spec.RetentionPolicy = v1alpha1.RetentionPolicy{}

				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", bb.Name))
				_, err := f.CreateBackupBlueprint(bb)
				Expect(err).To(HaveOccurred())
			})
			It("Should fail for adding inappropriate BackupConfiguration Schedule in BackupBlueprint", func() {
				// Create Storage Secret for Minio
				secret := createBackendSecretForMinio()

				// Generate BackupBlueprint definition
				bb := f.BackupBlueprint(getRepositoryInfo(secret.Name))
				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", bb.Name))
				bb.Spec.Schedule = ""
				_, err := f.CreateBackupBlueprint(bb)
				Expect(err).To(HaveOccurred())
			})
			It("Should fail auto-backup for adding inappropriate BackupBlueprint annotation in ReplicationController", func() {
				// Create BackupBlueprint
				createBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))
				// Deploy a ReplicationController
				rc := deployReplicationController(fmt.Sprintf("rc-%s", f.App()))
				// Generate Sample Data
				generateSampleData(rc)

				// wrong BackupBlueprint Name set as annotations to ReplicationController
				annotations := map[string]string{
					v1beta1.KeyBackupBlueprint: framework.WrongBackupBlueprintName,
					v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
					v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
				}
				// Adding and Ensuring annotations to workload
				addAnnotationsToWorkload(annotations, rc)

				By("Will fail to get respective BackupBlueprint")
				getAnnotations := rc.GetAnnotations()
				_, err := f.GetBackupBlueprint(getAnnotations[v1beta1.KeyBackupBlueprint])
				Expect(err).To(HaveOccurred())
			})
			It("Should fail for adding inappropriate TargetPath/MountPath annotations in ReplicationController", func() {
				// Create BackupBlueprint
				bb := createBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))
				// Deploy a ReplicationController
				rc := deployReplicationController(fmt.Sprintf("rc-%s", f.App()))
				// Generate Sample Data
				generateSampleData(rc)

				// wrong BackupBlueprint Name set as annotations to ReplicationController
				annotations := map[string]string{
					v1beta1.KeyBackupBlueprint: bb.Name,
					v1beta1.KeyTargetPaths:     framework.WrongTargetPath,
					v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
				}
				// Adding and Ensuring annotations to workload
				addAnnotationsToWorkload(annotations, rc)
				// check Stash Object status
				checkBackupSessionFailed(rc)
			})

		})
	})

})
