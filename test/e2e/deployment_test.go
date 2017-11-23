package e2e_test

import (
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
		restic     api.Restic
		cred       core.Secret
		deployment apps.Deployment
		recovery   api.Recovery
	)

	BeforeEach(func() {
		f = root.Invoke()
	})
	JustBeforeEach(func() {
		if missing, _ := BeZero().Match(cred); missing {
			Skip("Missing repository credential")
		}
		restic.Spec.Backend.StorageSecretName = cred.Name
		deployment = f.Deployment()
	})

	var (
		shouldBackupNewDeployment = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldBackupExistingDeployment = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldStopBackup = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Deleting restic " + restic.Name)
			f.DeleteRestic(restic.ObjectMeta)

			f.EventuallyDeployment(deployment.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldStopBackupIfLabelChanged = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Removing labels of Deployment " + deployment.Name)
			_, err = apps_util.PatchDeployment(f.KubeClient, &deployment, func(in *apps.Deployment) *apps.Deployment {
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

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Change selector of Restic " + restic.Name)
			err = f.UpdateRestic(restic.ObjectMeta, func(in *api.Restic) *api.Restic {
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

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			deployment.Spec.Replicas = types.Int32P(2) // two replicas
			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			f.CheckLeaderElection(deployment.ObjectMeta, api.KindDeployment)

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldInitializeAndBackupDeployment = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
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
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldOfflineBackupNewDeployment = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			/*cronJobName := "stash-backup-" + restic.Name
			By("Checking cron job created: " + cronJobName)
			Eventually(func() error {
				_, err := f.KubeClient.BatchV1beta1().CronJobs(restic.Namespace).Get(cronJobName, metav1.GetOptions{})
				return err
			}).Should(BeNil())*/

			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for init-container")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveInitContainer(util.StashContainer))

			By("Waiting for initial backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Waiting for next backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 2)))

			By("Waiting for backup event")
			f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">", 1)))
		}
	)

	Describe("Creating restic for", func() {
		/*AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})*/

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			FIt(`should backup new Deployment`, shouldBackupNewDeployment)
			It(`should backup existing Deployment`, shouldBackupExistingDeployment)
		})

		Context(`"S3" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForS3Backend()
				restic = f.ResticForS3Backend()
			})
			It(`should backup new Deployment`, shouldBackupNewDeployment)
			It(`should backup existing Deployment`, shouldBackupExistingDeployment)
		})

		Context(`"DO" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForDOBackend()
				restic = f.ResticForDOBackend()
			})
			It(`should backup new Deployment`, shouldBackupNewDeployment)
			It(`should backup existing Deployment`, shouldBackupExistingDeployment)
		})

		Context(`"GCS" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForGCSBackend()
				restic = f.ResticForGCSBackend()
			})
			It(`should backup new Deployment`, shouldBackupNewDeployment)
			It(`should backup existing Deployment`, shouldBackupExistingDeployment)
		})

		Context(`"Azure" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForAzureBackend()
				restic = f.ResticForAzureBackend()
			})
			It(`should backup new Deployment`, shouldBackupNewDeployment)
			It(`should backup existing Deployment`, shouldBackupExistingDeployment)
		})

		Context(`"Swift" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForSwiftBackend()
				restic = f.ResticForSwiftBackend()
			})
			It(`should backup new Deployment`, shouldBackupNewDeployment)
			It(`should backup existing Deployment`, shouldBackupExistingDeployment)
		})
	})

	Describe("Changing Deployment labels", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})
		BeforeEach(func() {
			cred = f.SecretForLocalBackend()
			restic = f.ResticForLocalBackend()
		})
		It(`should stop backup`, shouldStopBackupIfLabelChanged)
	})

	Describe("Changing Restic selector", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})
		BeforeEach(func() {
			cred = f.SecretForLocalBackend()
			restic = f.ResticForLocalBackend()
		})
		It(`should stop backup`, shouldStopBackupIfSelectorChanged)
	})

	Describe("Deleting restic for", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"S3" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForS3Backend()
				restic = f.ResticForS3Backend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"DO" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForDOBackend()
				restic = f.ResticForDOBackend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"GCS" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForGCSBackend()
				restic = f.ResticForGCSBackend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"Azure" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForAzureBackend()
				restic = f.ResticForAzureBackend()
			})
			It(`should stop backup`, shouldStopBackup)
		})

		Context(`"Swift" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForSwiftBackend()
				restic = f.ResticForSwiftBackend()
			})
			It(`should stop backup`, shouldStopBackup)
		})
	})

	Describe("Creating recovery for", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
			f.DeleteRecovery(recovery.ObjectMeta)
			framework.CleanupMinikubeHostPath()
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				recovery = f.RecoveryForRestic(restic.Name)
			})
			It(`should restore local deployment backup`, shouldRestoreDeployment)
		})

		Context(`"S3" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForS3Backend()
				restic = f.ResticForS3Backend()
				recovery = f.RecoveryForRestic(restic.Name)
			})
			It(`should restore s3 deployment backup`, shouldRestoreDeployment)
		})
	})

	Describe("Recovery as job's owner-ref", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
			f.DeleteRecovery(recovery.ObjectMeta)
			framework.CleanupMinikubeHostPath()
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				recovery = f.RecoveryForRestic(restic.Name)
			})
			It(`should delete job after recovery deleted`, func() {
				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating recovery " + recovery.Name)
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Checking Job exists")
				Eventually(func() bool {
					_, err := f.KubeClient.BatchV1().Jobs(recovery.Namespace).Get("stash-"+recovery.Name, metav1.GetOptions{})
					return err != nil
				}, time.Minute*3, time.Second*2).Should(BeTrue())

				By("Deleting recovery " + recovery.Name)
				err = f.DeleteRecovery(recovery.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Checking Job deleted")
				Eventually(func() bool {
					_, err := f.KubeClient.BatchV1().Jobs(recovery.Namespace).Get("stash-"+recovery.Name, metav1.GetOptions{})
					return kerr.IsNotFound(err) || kerr.IsGone(err)
				}, time.Minute*3, time.Second*2).Should(BeTrue())
			})
		})
	})

	Describe("Leader election for", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should elect leader and backup new Deployment`, shouldElectLeaderAndBackupDeployment)
		})
	})

	Describe("Stash initializer for", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It("should initialize and backup new Deployment", shouldInitializeAndBackupDeployment)
		})
	})

	Describe("Offline backup for", func() {
		//AfterEach(func() {
		//	f.DeleteDeployment(deployment.ObjectMeta)
		//	f.DeleteRestic(restic.ObjectMeta)
		//	f.DeleteSecret(cred.ObjectMeta)
		//})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
				restic.Spec.Type = api.BackupOffline
				restic.Spec.Schedule = "*/5 * * * *"
			})
			It(`should backup new Deployment`, shouldOfflineBackupNewDeployment)
		})
	})
})
