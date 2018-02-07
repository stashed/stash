package e2e_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/appscode/go/types"
	apps_util "github.com/appscode/kutil/apps/v1beta1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	"github.com/appscode/stash/test/e2e/framework"
	. "github.com/appscode/stash/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1beta1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Deployment", func() {
	var (
		err        error
		f          *framework.Invocation
		backup     api.Backup
		cred       core.Secret
		deployment apps.Deployment
		recovery   api.Recovery
	)

	BeforeEach(func() {
		f = root.Invoke()
	})
	AfterEach(func() {
		time.Sleep(60 * time.Second)
	})
	JustBeforeEach(func() {
		if missing, _ := BeZero().Match(cred); missing {
			Skip("Missing repository credential")
		}
		backup.Spec.Backend.StorageSecretName = cred.Name
		recovery.Spec.Backend.StorageSecretName = cred.Name
		deployment = f.Deployment()
	})

	var (
		shouldBackupNewDeployment = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating backup " + backup.Name)
			err = f.CreateBackup(backup)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyBackup(backup.ObjectMeta).Should(WithTransform(func(r *api.Backup) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			f.EventualEvent(backup.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldBackupExistingDeployment = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Creating backup " + backup.Name)
			err = f.CreateBackup(backup)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyBackup(backup.ObjectMeta).Should(WithTransform(func(r *api.Backup) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			f.EventualEvent(backup.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldStopBackup = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating backup " + backup.Name)
			err = f.CreateBackup(backup)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyBackup(backup.ObjectMeta).Should(WithTransform(func(r *api.Backup) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Deleting backup " + backup.Name)
			f.DeleteBackup(backup.ObjectMeta)

			By("Waiting to remove sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldStopBackupIfLabelChanged = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating backup " + backup.Name)
			err = f.CreateBackup(backup)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyBackup(backup.ObjectMeta).Should(WithTransform(func(r *api.Backup) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Removing labels of Deployment " + deployment.Name)
			_, _, err = apps_util.PatchDeployment(f.KubeClient, &deployment, func(in *apps.Deployment) *apps.Deployment {
				in.Labels = map[string]string{
					"app": "unmatched",
				}
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			f.EventuallyDeployment(deployment.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldStopBackupIfSelectorChanged = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating backup " + backup.Name)
			err = f.CreateBackup(backup)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyBackup(backup.ObjectMeta).Should(WithTransform(func(r *api.Backup) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Change selector of Backup " + backup.Name)
			err = f.UpdateBackup(backup.ObjectMeta, func(in *api.Backup) *api.Backup {
				in.Spec.Selector = metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "unmatched",
					},
				}
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			f.EventuallyDeployment(deployment.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldRestoreDeployment = func() {
			shouldBackupNewDeployment()

			recovery.Spec.Workload = api.LocalTypedReference{
				Kind: api.KindDeployment,
				Name: deployment.Name,
			}

			By("Creating recovery " + recovery.Name)
			err = f.CreateRecovery(recovery)
			Expect(err).NotTo(HaveOccurred())

			f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())
		}

		shouldElectLeaderAndBackupDeployment = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating backup " + backup.Name)
			err = f.CreateBackup(backup)
			Expect(err).NotTo(HaveOccurred())

			deployment.Spec.Replicas = types.Int32P(2) // two replicas
			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			f.CheckLeaderElection(deployment.ObjectMeta, api.KindDeployment)

			By("Waiting for backup to complete")
			f.EventuallyBackup(backup.ObjectMeta).Should(WithTransform(func(r *api.Backup) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			f.EventualEvent(backup.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldInitializeAndBackupDeployment = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating backup " + backup.Name)
			err = f.CreateBackup(backup)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			obj, err := f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			// By("Waiting for sidecar")
			// f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			// sidecar should be added as soon as deployment created, we don't need to wait for it
			By("Checking sidecar created")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyBackup(backup.ObjectMeta).Should(WithTransform(func(r *api.Backup) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			f.EventualEvent(backup.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldDeleteJobAndDependents = func(jobName, namespace string) {
			By("Checking Job deleted")
			Eventually(func() bool {
				_, err := f.KubeClient.BatchV1().Jobs(recovery.Namespace).Get(jobName, metav1.GetOptions{})
				return kerr.IsNotFound(err) || kerr.IsGone(err)
			}, time.Minute*3, time.Second*2).Should(BeTrue())

			By("Checking pods deleted")
			Eventually(func() bool {
				pods, err := f.KubeClient.CoreV1().Pods(recovery.Namespace).List(metav1.ListOptions{
					LabelSelector: "job-name=" + jobName, // pods created by job has a job-name label
				})
				Expect(err).NotTo(HaveOccurred())
				return len(pods.Items) == 0
			}, time.Minute*3, time.Second*2).Should(BeTrue())

			By("Checking service-account deleted")
			Eventually(func() bool {
				_, err := f.KubeClient.CoreV1().ServiceAccounts(recovery.Namespace).Get(jobName, metav1.GetOptions{})
				return kerr.IsNotFound(err) || kerr.IsGone(err)
			}, time.Minute*3, time.Second*2).Should(BeTrue())

			By("Checking role-binding deleted")
			Eventually(func() bool {
				_, err := f.KubeClient.RbacV1().RoleBindings(recovery.Namespace).Get(jobName, metav1.GetOptions{})
				return kerr.IsNotFound(err) || kerr.IsGone(err)
			}, time.Minute*3, time.Second*2).Should(BeTrue())
		}
	)

	Describe("Creating backup for", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteBackup(backup.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				backup = f.BackupForLocalBackend()
			})
			It(`should backup new Deployment`, shouldBackupNewDeployment)
			It(`should backup existing Deployment`, shouldBackupExistingDeployment)
		})

		Context(`"S3" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForS3Backend()
				backup = f.BackupForS3Backend()
			})
			It(`should backup new Deployment`, shouldBackupNewDeployment)
			It(`should backup existing Deployment`, shouldBackupExistingDeployment)
		})

		Context(`"DO" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForDOBackend()
				backup = f.BackupForDOBackend()
			})
			It(`should backup new Deployment`, shouldBackupNewDeployment)
			It(`should backup existing Deployment`, shouldBackupExistingDeployment)
		})

		Context(`"GCS" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForGCSBackend()
				backup = f.BackupForGCSBackend()
			})
			It(`should backup new Deployment`, shouldBackupNewDeployment)
			It(`should backup existing Deployment`, shouldBackupExistingDeployment)
		})

		Context(`"Azure" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForAzureBackend()
				backup = f.BackupForAzureBackend()
			})
			It(`should backup new Deployment`, shouldBackupNewDeployment)
			It(`should backup existing Deployment`, shouldBackupExistingDeployment)
		})

		Context(`"Swift" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForSwiftBackend()
				backup = f.BackupForSwiftBackend()
			})
			It(`should backup new Deployment`, shouldBackupNewDeployment)
			It(`should backup existing Deployment`, shouldBackupExistingDeployment)
		})

		Context(`"B2" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForB2Backend()
				backup = f.BackupForB2Backend()
			})
			It(`should backup new Deployment`, shouldBackupNewDeployment)
			It(`should backup existing Deployment`, shouldBackupExistingDeployment)
		})
	})

	Describe("Changing Deployment labels", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteBackup(backup.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})
		BeforeEach(func() {
			cred = f.SecretForLocalBackend()
			backup = f.BackupForLocalBackend()
		})
		It(`should stop backup`, shouldStopBackupIfLabelChanged)
	})

	Describe("Changing Backup selector", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteBackup(backup.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})
		BeforeEach(func() {
			cred = f.SecretForLocalBackend()
			backup = f.BackupForLocalBackend()
		})
		It(`should stop backup`, shouldStopBackupIfSelectorChanged)
	})

	Describe("Deleting backup for", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				backup = f.BackupForLocalBackend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"S3" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForS3Backend()
				backup = f.BackupForS3Backend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"DO" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForDOBackend()
				backup = f.BackupForDOBackend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"GCS" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForGCSBackend()
				backup = f.BackupForGCSBackend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"Azure" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForAzureBackend()
				backup = f.BackupForAzureBackend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"Swift" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForSwiftBackend()
				backup = f.BackupForSwiftBackend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"B2" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForB2Backend()
				backup = f.BackupForB2Backend()
			})
			It(`should stop backup`, shouldStopBackup)
		})
	})

	Describe("Creating recovery for", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteBackup(backup.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
			f.DeleteRecovery(recovery.ObjectMeta)
			framework.CleanupMinikubeHostPath()
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				backup = f.BackupForHostPathLocalBackend()
				recovery = f.RecoveryForBackup(backup)
			})
			It(`should restore local deployment backup and cleanup dependents`, func() {
				By("Checking recovery successful")
				shouldRestoreDeployment()
				By("Checking cleanup")
				shouldDeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, recovery.Namespace)
			})
		})

		Context(`"S3" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForS3Backend()
				backup = f.BackupForS3Backend()
				recovery = f.RecoveryForBackup(backup)
			})
			It(`should restore s3 deployment backup`, shouldRestoreDeployment)
		})
	})

	Describe("Recovery as job's owner-ref", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteBackup(backup.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
			f.DeleteRecovery(recovery.ObjectMeta)
			framework.CleanupMinikubeHostPath()
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				backup = f.BackupForHostPathLocalBackend()
				recovery = f.RecoveryForBackup(backup)
			})
			It(`should delete job after recovery deleted`, func() {
				recovery.Spec.Workload = api.LocalTypedReference{
					Kind: api.KindDeployment,
					Name: deployment.Name,
				}

				By("Creating recovery " + recovery.Name)
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				jobName := util.RecoveryJobPrefix + recovery.Name

				By("Checking Job exists")
				Eventually(func() bool {
					_, err := f.KubeClient.BatchV1().Jobs(recovery.Namespace).Get(jobName, metav1.GetOptions{})
					return err == nil
				}, time.Minute*3, time.Second*2).Should(BeTrue())

				By("Deleting recovery " + recovery.Name)
				err = f.DeleteRecovery(recovery.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Checking cleanup")
				shouldDeleteJobAndDependents(jobName, recovery.Namespace)
			})
		})
	})

	Describe("Leader election for", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteBackup(backup.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				backup = f.BackupForLocalBackend()
			})
			It(`should elect leader and backup new Deployment`, shouldElectLeaderAndBackupDeployment)
		})
	})

	Describe("Stash initializer for", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteBackup(backup.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				backup = f.BackupForLocalBackend()
			})
			It("should initialize and backup new Deployment", shouldInitializeAndBackupDeployment)
		})
	})

	Describe("Offline backup for", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteBackup(backup.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
			framework.CleanupMinikubeHostPath()
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				backup = f.BackupForHostPathLocalBackend()
				backup.Spec.Type = api.BackupOffline
				backup.Spec.Schedule = "*/5 * * * *"
			})
			It(`should backup new Deployment`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating backup " + backup.Name)
				err = f.CreateBackup(backup)
				Expect(err).NotTo(HaveOccurred())

				cronJobName := util.KubectlCronPrefix + backup.Name
				By("Checking cron job created: " + cronJobName)
				Eventually(func() error {
					_, err := f.KubeClient.BatchV1beta1().CronJobs(backup.Namespace).Get(cronJobName, metav1.GetOptions{})
					return err
				}).Should(BeNil())

				By("Creating Deployment " + deployment.Name)
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for init-container")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveInitContainer(util.StashContainer))

				By("Waiting for initial backup to complete")
				f.EventuallyBackup(backup.ObjectMeta).Should(WithTransform(func(r *api.Backup) int64 {
					return r.Status.BackupCount
				}, BeNumerically(">=", 1)))

				By("Waiting for next backup to complete")
				f.EventuallyBackup(backup.ObjectMeta).Should(WithTransform(func(r *api.Backup) int64 {
					return r.Status.BackupCount
				}, BeNumerically(">=", 2)))

				By("Waiting for backup event")
				f.EventualEvent(backup.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">", 1)))
			})
		})
	})

	Describe("No retention policy", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteBackup(backup.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				backup = f.BackupForLocalBackend()
				backup.Spec.FileGroups[0].RetentionPolicyName = ""
				backup.Spec.RetentionPolicies = []api.RetentionPolicy{}
			})
			It(`should backup new Deployment`, shouldBackupNewDeployment)
		})
	})
	Describe("Minio server", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteBackup(backup.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
			f.DeleteMinioServer()
		})
		Context("With cacert", func() {
			BeforeEach(func() {
				By("Creating Minio server with cacert")
				addrs, err := f.CreateMinioServer()
				Expect(err).NotTo(HaveOccurred())

				backup = f.BackupForMinioBackend("https://" + addrs)
				cred = f.SecretForMinioBackend(true)

			})

			It("Should backup new Deployment", func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating backup")
				err = f.CreateBackup(backup)
				Expect(err).NotTo(HaveOccurred())

				By("Creating Deployment " + deployment.Name)
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for backup to complete")
				f.EventuallyBackup(backup.ObjectMeta).Should(WithTransform(func(r *api.Backup) int64 {
					return r.Status.BackupCount
				}, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				f.EventualEvent(backup.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

			})
		})
		Context("Without cacert", func() {
			BeforeEach(func() {
				By("Creating Minio server with cacert")
				addrs, err := f.CreateMinioServer()
				Expect(err).NotTo(HaveOccurred())

				backup = f.BackupForMinioBackend("https://" + addrs)
				cred = f.SecretForMinioBackend(false)

			})

			It("Should fail to backup new Deployment", func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating backup without cacert")
				err = f.CreateBackup(backup)
				Expect(err).NotTo(HaveOccurred())

				By("Creating Deployment " + deployment.Name)
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting to count failed setup event")
				f.EventualWarning(backup.ObjectMeta).Should(WithTransform(f.CountFailedSetup, BeNumerically(">=", 1)))

				By("Waiting to count successful backup event")
				f.EventualEvent(backup.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically("==", 0)))

			})
		})
	})

	Describe("Private docker registry", func() {
		var registryCred core.Secret
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteBackup(backup.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
			f.DeleteSecret(registryCred.ObjectMeta)
		})
		BeforeEach(func() {
			By("Reading docker config json file")
			dockerCfgJson, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".docker/config.json"))
			Expect(err).NotTo(HaveOccurred())

			registryCred = f.SecretForRegistry(dockerCfgJson)
			By("Creating registry Secret " + registryCred.Name)
			err = f.CreateSecret(registryCred)
			Expect(err).NotTo(HaveOccurred())

			cred = f.SecretForLocalBackend()
			backup = f.BackupForLocalBackend()
			backup.Spec.ImagePullSecrets = []core.LocalObjectReference{
				{
					Name: registryCred.Name,
				},
			}
		})
		It(`should backup new Deployment`, shouldBackupNewDeployment)
	})

	Describe("Pause Backup to stop backup", func() {
		Context(`"Local" backend`, func() {
			AfterEach(func() {
				f.DeleteDeployment(deployment.ObjectMeta)
				f.DeleteBackup(backup.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
			})
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				backup = f.BackupForLocalBackend()
			})
			It(`should be able to Pause and Resume backup`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating backup")
				err = f.CreateBackup(backup)
				Expect(err).NotTo(HaveOccurred())

				By("Creating Deployment " + deployment.Name)
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for backup to complete")
				f.EventuallyBackup(backup.ObjectMeta).Should(WithTransform(func(r *api.Backup) int64 {
					return r.Status.BackupCount
				}, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				f.EventualEvent(backup.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By(`Patching Backup with "paused: true"`)
				err = f.CreateOrPatchBackup(backup.ObjectMeta, func(in *api.Backup) *api.Backup {
					in.Spec.Paused = true
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				backupObj, err := f.StashClient.StashV1alpha1().Backups(backup.Namespace).Get(backup.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				previousBackupCount := backupObj.Status.BackupCount

				By("Wating 2 minutes")
				time.Sleep(2 * time.Minute)

				By("Checking that Backup count has not changed")
				backupObj, err = f.StashClient.StashV1alpha1().Backups(backup.Namespace).Get(backup.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(backupObj.Status.BackupCount).Should(BeNumerically("==", previousBackupCount))

				By(`Patching Backup with "paused: false"`)
				err = f.CreateOrPatchBackup(backup.ObjectMeta, func(in *api.Backup) *api.Backup {
					in.Spec.Paused = false
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for backup to complete")
				f.EventuallyBackup(backup.ObjectMeta).Should(WithTransform(func(r *api.Backup) int64 {
					return r.Status.BackupCount
				}, BeNumerically(">", previousBackupCount)))

				By("Waiting for backup event")
				f.EventualEvent(backup.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">", previousBackupCount)))

			})

		})
	})
})
