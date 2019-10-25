package auto_backup

import (
	"fmt"

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

		generateSampleData = func(ss *apps.StatefulSet) sets.String {
			By("Generating sample data inside StatefulSet pods")
			err := f.CreateSampleDataInsideWorkload(ss.ObjectMeta, apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that sample data has been generated")
			sampleData, err := f.ReadSampleDataFromFromWorkload(ss.ObjectMeta, apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleData).ShouldNot(BeEmpty())

			return sampleData
		}

		addAnnotationsToTarget = func(annotations map[string]string, ss *apps.StatefulSet) {
			By(fmt.Sprintf("Adding auto-backup specific annotations to the StatefulSet: %s/%s", ss.Namespace, ss.Name))
			err := f.AddAutoBackupAnnotationsToTarget(annotations, ss)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the auto-backup annotations has been added successfully")
			f.EventuallyAutoBackupAnnotationsFound(annotations, ss).Should(BeTrue())
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

		checkRepositoryAndBackupConfiguration = func(ss *apps.StatefulSet) *v1beta1.BackupConfiguration {
			// BackupBlueprint create BackupConfiguration and Repository such that
			// the name of the BackupConfiguration and Repository will follow
			// the patter: <lower case of the workload kind>-<workload name>.
			// we will form the meta name and namespace for farther process.
			objMeta := metav1.ObjectMeta{
				Name:      fmt.Sprintf("statefulset-%s", ss.Name),
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
			f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Deployment to be ready with sidecar")
			err = f.WaitUntilStatefulSetReadyWithSidecar(ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			return backupConfig
		}
	)

	Context("StatefulSet", func() {

		Context("Success Case", func() {

			It("should success auto-backup for the StatefulSet", func() {
				// Create BackupBlueprint
				bb := createBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

				// Deploy a StatefulSet
				ss := deploySS(fmt.Sprintf("ss-%s", f.App()))

				// Generate Sample Data
				generateSampleData(ss)

				// create annotations for StatefulSet
				annotations := map[string]string{
					v1beta1.KeyBackupBlueprint: bb.Name,
					v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
					v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
				}
				// Adding and Ensuring annotations to Target
				addAnnotationsToTarget(annotations, ss)

				// ensure Repository and BackupConfiguration
				backupConfig := checkRepositoryAndBackupConfiguration(ss)

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

					// Deploy a StatefulSet
					ss := deploySS(fmt.Sprintf("ss-%s", f.App()))

					// Generate Sample Data
					generateSampleData(ss)

					// create annotations for StatefulSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					addAnnotationsToTarget(annotations, ss)

					// ensure Repository and BackupConfiguration
					backupConfig := checkRepositoryAndBackupConfiguration(ss)

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

					// Deploy a StatefulSet
					ss := deploySS(fmt.Sprintf("ss-%s", f.App()))

					// Generate Sample Data
					generateSampleData(ss)

					// create annotations for StatefulSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					addAnnotationsToTarget(annotations, ss)

					// ensure Repository and BackupConfiguration
					backupConfig := checkRepositoryAndBackupConfiguration(ss)

					// Take an Instant Backup the Sample Data
					instantBackupFailed(backupConfig)
				})
			})

			Context("Add inappropriate annotation to Target", func() {
				It("Should fail auto-backup for adding inappropriate BackupBlueprint annotation in StatefulSet", func() {
					// Create BackupBlueprint
					createBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

					// Deploy a StatefulSet
					ss := deploySS(fmt.Sprintf("ss-%s", f.App()))

					// Generate Sample Data
					generateSampleData(ss)

					// set wrong annotations to StatefulSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: framework.WrongBackupBlueprintName,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					addAnnotationsToTarget(annotations, ss)

					By("Will fail to get respective BackupBlueprint")
					getAnnotations := ss.GetAnnotations()
					_, err := f.GetBackupBlueprint(getAnnotations[v1beta1.KeyBackupBlueprint])
					Expect(err).To(HaveOccurred())
				})
				It("should fail BackupSession for adding inappropriate TargetPath/MountPath StatefulSet", func() {
					// Create BackupBlueprint
					bb := createBackupBlueprint(fmt.Sprintf("backupblueprint-%s", f.App()))

					// Deploy a StatefulSet
					ss := deploySS(fmt.Sprintf("ss-%s", f.App()))

					// Generate Sample Data
					generateSampleData(ss)

					// set wrong annotations to StatefulSet
					annotations := map[string]string{
						v1beta1.KeyBackupBlueprint: bb.Name,
						v1beta1.KeyTargetPaths:     framework.WrongTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					}
					// Adding and Ensuring annotations to Target
					addAnnotationsToTarget(annotations, ss)

					// ensure Repository and BackupConfiguration
					backupConfig := checkRepositoryAndBackupConfiguration(ss)

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
