package auto_backup

import (
	"fmt"
	"strings"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	"github.com/appscode/go/sets"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		createBackendSecretForMinio = func() *core.Secret {
			// Create Storage Secret
			cred := f.SecretForMinioBackend(true)

			if missing, _ := BeZero().Match(cred); missing {
				Skip("Missing Minio credential")
			}
			By(fmt.Sprintf("Creating Storage Secret for Minio: %s/%s", cred.Namespace, cred.Name))
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

			By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
			createdBB, err := f.CreateBackupBlueprint(bb)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdBB)
			return createdBB
		}

		createPVC = func(name string) *core.PersistentVolumeClaim {
			// Generate PVC definition
			pvc := f.PersistentVolumeClaim()
			pvc.Name = fmt.Sprintf("%s-pvc-%s", strings.Split(name, "-")[0], f.App())

			By(fmt.Sprintf("Creating PVC: %s/%s", pvc.Namespace, pvc.Name))
			createdPVC, err := f.CreatePersistentVolumeClaim(pvc)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdPVC)

			return createdPVC
		}

		deployReplicaSet = func(name string) *apps.ReplicaSet {
			// Create PVC for ReplicaSet
			pvc := createPVC(name)
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

		generateSampleData = func(rs *apps.ReplicaSet) sets.String {
			By("Generating sample data inside ReplicaSet pods")
			err := f.CreateSampleDataInsideWorkload(rs.ObjectMeta, apis.KindReplicaSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that sample data has been generated")
			sampleData, err := f.ReadSampleDataFromFromWorkload(rs.ObjectMeta, apis.KindReplicaSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleData).ShouldNot(BeEmpty())

			return sampleData
		}

		addAnnotationsToTarget = func(annotations map[string]string, rs *apps.ReplicaSet) {
			By(fmt.Sprintf("Adding auto-backup specific annotations to the ReplicaSet: %s/%s", rs.Namespace, rs.Name))
			err := f.AddAutoBackupAnnotationsToTarget(annotations, rs)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the auto-backup annotations has been added successfully")
			f.EventuallyAutoBackupAnnotationsFound(annotations, rs).Should(BeTrue())
		}

		takeInstantBackup = func(backupConfig *v1beta1.BackupConfiguration) {
			// Trigger Instant Backup
			By("Triggering Instant Backup")
			backupSession, err := f.TriggerInstantBackup(backupConfig)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(backupSession)

			By("Waiting for backup process to complete")
			f.EventuallyBackupProcessCompleted(backupSession.ObjectMeta).Should(BeTrue())

			By("Verifying that BackupSession has succeeded")
			completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
		}

		instantBackupFailed = func(backupConfig *v1beta1.BackupConfiguration) {
			// Trigger Instant Backup
			By("Triggering Instant Backup")
			backupSession, err := f.TriggerInstantBackup(backupConfig)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(backupSession)

			By("Waiting for backup process to complete")
			f.EventuallyBackupProcessCompleted(backupSession.ObjectMeta).Should(BeTrue())

			By("Verifying that BackupSession has failed")
			completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))
		}

		checkRepositoryAndBackupConfiguration = func(rs *apps.ReplicaSet) *v1beta1.BackupConfiguration {
			// BackupBlueprint create BackupConfiguration and Repository such that
			// the name of the BackupConfiguration and Repository will follow
			// the patter: <lower case of the workload kind>-<workload name>.
			// we will form the meta name and namespace for farther process.
			objMeta := metav1.ObjectMeta{
				Name:      fmt.Sprintf("replicaset-%s", rs.Name),
				Namespace: f.Namespace(),
			}
			By("Waiting for Repository")
			f.EventuallyRepositoryCreated(objMeta).Should(BeTrue())

			By("Waiting for BackupConfiguration")
			f.EventuallyBackupConfigurationCreated(objMeta).Should(BeTrue())
			backupConfig, err := f.StashClient.StashV1beta1().BackupConfigurations(objMeta.Namespace).Get(objMeta.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that backup triggering CronJob has been created")
			f.EventuallyCronJobCreated(objMeta).Should(BeTrue())

			By("Verifying that sidecar has been injected")
			f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for ReplicaSet to be ready with sidecar")
			err = f.WaitUntilRSReadyWithSidecar(rs.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			return backupConfig
		}
	)

	Context("ReplicaSet", func() {

		Context("Success Case", func() {

			It("should success auto-backup for the ReplicaSet", func() {
				// Create BackupBlueprint
				bb := createBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

				// Deploy a ReplicaSet
				rs := deployReplicaSet(fmt.Sprintf("rs-%s", f.App()))

				// Generate Sample Data
				generateSampleData(rs)

				// create annotations for ReplicaSet
				annotations := map[string]string{
					v1beta1.KeyBackupBlueprint: bb.Name,
					v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
					v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
				}
				// Adding and Ensuring annotations to Target
				addAnnotationsToTarget(annotations, rs)

				// ensure Repository and BackupConfiguration
				backupConfig := checkRepositoryAndBackupConfiguration(rs)

				// Take an Instant Backup the Sample Data
				takeInstantBackup(backupConfig)
			})
		})

		Context("Failure Case", func() {

			Context("Add inappropriate Repository and BackupConfiguration credential to BackupBlueprint", func() {
				It("should should fail BackupSession for missing Backend credential", func() {
					// Create Storage Secret for Minio
					secret := createBackendSecretForMinio()

					// Generate BackupBlueprint definition
					bb := f.BackupBlueprint(getRepositoryInfo(secret.Name))
					bb.Spec.Backend.S3 = &store.S3Spec{}
					By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
					_, err := f.CreateBackupBlueprint(bb)
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(bb)

					// Deploy a ReplicaSet
					rs := deployReplicaSet(fmt.Sprintf("rs-%s", f.App()))

					// Generate Sample Data
					generateSampleData(rs)

					// create annotations for ReplicaSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					addAnnotationsToTarget(annotations, rs)

					// ensure Repository and BackupConfiguration
					backupConfig := checkRepositoryAndBackupConfiguration(rs)

					instantBackupFailed(backupConfig)
				})
				It("should fail BackupSession for missing RetentionPolicy", func() {
					// Create Storage Secret for Minio
					secret := createBackendSecretForMinio()

					// Generate BackupBlueprint definition
					bb := f.BackupBlueprint(getRepositoryInfo(secret.Name))
					bb.Spec.RetentionPolicy = v1alpha1.RetentionPolicy{}

					By(fmt.Sprintf("Creating BackupBlueprint: %s", bb.Name))
					_, err := f.CreateBackupBlueprint(bb)
					Expect(err).NotTo(HaveOccurred())

					// Deploy a ReplicaSet
					rs := deployReplicaSet(fmt.Sprintf("rs-%s", f.App()))

					// Generate Sample Data
					generateSampleData(rs)

					// create annotations for ReplicaSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					addAnnotationsToTarget(annotations, rs)

					// ensure Repository and BackupConfiguration
					backupConfig := checkRepositoryAndBackupConfiguration(rs)

					instantBackupFailed(backupConfig)
				})
			})

			Context("Add inappropriate annotation to Target", func() {
				It("should fail to get respective BackupBlueprint for adding wrong BackupBlueprint name", func() {
					// Create BackupBlueprint
					createBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))
					// Deploy a ReplicaSet
					rs := deployReplicaSet(fmt.Sprintf("rs-%s", f.App()))
					// Generate Sample Data
					generateSampleData(rs)

					// set wrong annotations to ReplicaSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: framework.WrongBackupBlueprintName,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					addAnnotationsToTarget(annotations, rs)

					By("Will fail to get respective BackupBlueprint")
					getAnnotations := rs.GetAnnotations()
					_, err := f.GetBackupBlueprint(getAnnotations[v1beta1.KeyBackupBlueprint])
					Expect(err).To(HaveOccurred())
				})
				It("should fail BackupSession for adding inappropriate TargetPath/MountPath ReplicaSet", func() {
					// Create BackupBlueprint
					bb := createBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))
					// Deploy a ReplicaSet
					rs := deployReplicaSet(fmt.Sprintf("rs-%s", f.App()))
					// Generate Sample Data
					generateSampleData(rs)

					// set wrong annotations to ReplicaSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.WrongTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					addAnnotationsToTarget(annotations, rs)

					// ensure Repository and BackupConfiguration
					backupConfig := checkRepositoryAndBackupConfiguration(rs)

					// Trigger Instant Backup
					By("Triggering Instant Backup")
					backupSession, err := f.TriggerInstantBackup(backupConfig)
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(backupSession)

					By("Waiting for backup process to complete")
					f.EventuallyBackupProcessCompleted(backupSession.ObjectMeta).Should(BeTrue())

					By("Verifying that BackupSession has failed")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))
				})

			})
		})
	})

})
