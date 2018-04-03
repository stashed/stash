package e2e_test

import (
	"time"

	"github.com/appscode/go/types"
	core_util "github.com/appscode/kutil/core/v1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	"github.com/appscode/stash/test/e2e/framework"
	. "github.com/appscode/stash/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ReplicationController", func() {
	var (
		err          error
		f            *framework.Invocation
		restic       api.Restic
		secondRestic api.Restic
		cred         core.Secret
		rc           core.ReplicationController
		recovery     api.Recovery
	)

	BeforeEach(func() {
		f = root.Invoke()
	})
	AfterEach(func() {
		f.DeleteRepositories()
		time.Sleep(60 * time.Second)
	})
	JustBeforeEach(func() {
		if missing, _ := BeZero().Match(cred); missing {
			Skip("Missing repository credential")
		}
		restic.Spec.Backend.StorageSecretName = cred.Name
		secondRestic.Spec.Backend.StorageSecretName = cred.Name
		recovery.Spec.Backend.StorageSecretName = cred.Name
		rc = f.ReplicationController()
	})

	var (
		shouldBackupNewReplicationController = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicationController " + rc.Name)
			_, err = f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(repos.Items).NotTo(BeEmpty())
			f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldBackupExistingReplicationController = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicationController " + rc.Name)
			_, err = f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(repos.Items).NotTo(BeEmpty())
			f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldStopBackup = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicationController " + rc.Name)
			_, err = f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Deleting restic " + restic.Name)
			f.DeleteRestic(restic.ObjectMeta)

			By("Waiting to remove sidecar")
			f.EventuallyReplicationController(rc.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldStopBackupIfLabelChanged = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicationController " + rc.Name)
			_, err = f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Removing labels of ReplicationController " + rc.Name)
			_, _, err = core_util.PatchRC(f.KubeClient, &rc, func(in *core.ReplicationController) *core.ReplicationController {
				in.Labels = map[string]string{
					"app": "unmatched",
				}
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			f.EventuallyReplicationController(rc.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldStopBackupIfSelectorChanged = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicationController " + rc.Name)
			_, err = f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

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

			f.EventuallyReplicationController(rc.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldElectLeaderAndBackupRC = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			rc.Spec.Replicas = types.Int32P(2) // two replicas
			By("Creating ReplicationController " + rc.Name)
			_, err = f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())

			f.CheckLeaderElection(rc.ObjectMeta, api.KindReplicationController)

			By("Waiting for sidecar")
			f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(repos.Items).NotTo(BeEmpty())
			f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldMutateAndBackupNewReplicationController = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicationController " + rc.Name)
			obj, err := f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())

			// sidecar should be added as soon as rc created, we don't need to wait for it
			By("Checking sidecar created")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(repos.Items).NotTo(BeEmpty())
			f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldNotMutateNewReplicationController = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicationController " + rc.Name)
			obj, err := f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar not added")
			Expect(obj).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldRejectToCreateNewReplicationController = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating first restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating second restic " + secondRestic.Name)
			err = f.CreateRestic(secondRestic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicationController " + rc.Name)
			_, err := f.CreateReplicationController(rc)
			Expect(err).To(HaveOccurred())
		}

		shouldRemoveSidecarInstantly = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicationController " + rc.Name)
			obj, err := f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar added")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Removing labels of ReplicationController " + rc.Name)
			obj, _, err = core_util.PatchRC(f.KubeClient, &rc, func(in *core.ReplicationController) *core.ReplicationController {
				in.Labels = map[string]string{
					"app": "unmatched",
				}
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar has removed")
			Expect(obj).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldAddSidecarInstantly = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicationController " + rc.Name)
			previousLabel := rc.Labels
			rc.Labels = map[string]string{
				"app": "unmatched",
			}
			obj, err := f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar not added")
			Expect(obj).ShouldNot(HaveSidecar(util.StashContainer))

			By("Adding label to match restic" + rc.Name)
			obj, _, err = core_util.PatchRC(f.KubeClient, &rc, func(in *core.ReplicationController) *core.ReplicationController {
				in.Labels = previousLabel
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar added")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
		}
	)

	Describe("Creating restic for", func() {
		AfterEach(func() {
			f.DeleteReplicationController(rc.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should backup new ReplicationController`, shouldBackupNewReplicationController)
			It(`should backup existing ReplicationController`, shouldBackupExistingReplicationController)
		})

		Context(`"S3" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForS3Backend()
				restic = f.ResticForS3Backend()
			})
			It(`should backup new ReplicationController`, shouldBackupNewReplicationController)
			It(`should backup existing ReplicationController`, shouldBackupExistingReplicationController)
		})

		Context(`"DO" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForDOBackend()
				restic = f.ResticForDOBackend()
			})
			It(`should backup new ReplicationController`, shouldBackupNewReplicationController)
			It(`should backup existing ReplicationController`, shouldBackupExistingReplicationController)
		})

		Context(`"GCS" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForGCSBackend()
				restic = f.ResticForGCSBackend()
			})
			It(`should backup new ReplicationController`, shouldBackupNewReplicationController)
			It(`should backup existing ReplicationController`, shouldBackupExistingReplicationController)
		})

		Context(`"Azure" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForAzureBackend()
				restic = f.ResticForAzureBackend()
			})
			It(`should backup new ReplicationController`, shouldBackupNewReplicationController)
			It(`should backup existing ReplicationController`, shouldBackupExistingReplicationController)
		})

		Context(`"Swift" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForSwiftBackend()
				restic = f.ResticForSwiftBackend()
			})
			It(`should backup new ReplicationController`, shouldBackupNewReplicationController)
			It(`should backup existing ReplicationController`, shouldBackupExistingReplicationController)
		})

		Context(`"B2" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForB2Backend()
				restic = f.ResticForB2Backend()
			})
			It(`should backup new ReplicationController`, shouldBackupNewReplicationController)
			It(`should backup existing ReplicationController`, shouldBackupExistingReplicationController)
		})
	})

	Describe("Changing ReplicationController labels", func() {
		AfterEach(func() {
			f.DeleteReplicationController(rc.ObjectMeta)
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
			f.DeleteReplicationController(rc.ObjectMeta)
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
			f.DeleteReplicationController(rc.ObjectMeta)
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

		Context(`"B2" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForB2Backend()
				restic = f.ResticForB2Backend()
			})
			It(`should stop backup`, shouldStopBackup)
		})
	})

	Describe("Leader election for", func() {
		AfterEach(func() {
			f.DeleteReplicationController(rc.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should elect leader and backup new RC`, shouldElectLeaderAndBackupRC)
		})
	})

	Describe("Stash Webhook for", func() {
		BeforeEach(func() {
			if !f.WebhookEnabled {
				Skip("Webhook is disabled")
			}
		})
		AfterEach(func() {
			f.DeleteReplicationController(rc.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
			f.DeleteRestic(secondRestic.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
				secondRestic = restic
				secondRestic.Name = "second-restic"
			})
			It("should mutate and backup new ReplicationController", shouldMutateAndBackupNewReplicationController)
			It("should not mutate new ReplicationController if no restic select it", shouldNotMutateNewReplicationController)
			It("should reject to create new ReplicationController if multiple restic select it", shouldRejectToCreateNewReplicationController)
			It("should remove sidecar instantly if label change to match no restic", shouldRemoveSidecarInstantly)
			It("should add sidecar instantly if label change to match single restic", shouldAddSidecarInstantly)
		})
	})

	Describe("Offline backup for", func() {
		AfterEach(func() {
			f.DeleteReplicationController(rc.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
			framework.CleanupMinikubeHostPath()
		})

		Context(`Single Replica`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				restic.Spec.Type = api.BackupOffline
				restic.Spec.Schedule = "*/5 * * * *"
			})
			It(`should backup new RC`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating ReplicationController " + rc.Name)
				_, err = f.CreateReplicationController(rc)
				Expect(err).NotTo(HaveOccurred())

				cronJobName := util.ScaledownCronPrefix + restic.Name
				By("Checking cron job created: " + cronJobName)
				Eventually(func() error {
					_, err := f.KubeClient.BatchV1beta1().CronJobs(restic.Namespace).Get(cronJobName, metav1.GetOptions{})
					return err
				}).Should(BeNil())

				By("Waiting for scale down replication controller to 0 replica")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveReplica(0))

				By("Waiting for scale up replication controller to 1 replica")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveReplica(1))

				By("Waiting for init-container")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveInitContainer(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(repos.Items).NotTo(BeEmpty())
				f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Waiting for scale up replication controller to original replica")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveReplica(int(*rc.Spec.Replicas)))
			})
		})

		Context("Multiple Replica", func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				restic.Spec.Type = api.BackupOffline
				restic.Spec.Schedule = "*/5 * * * *"
			})
			It(`should backup new Replication Controller`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating replication controller " + rc.Name)
				rc.Spec.Replicas = types.Int32P(3)
				_, err = f.CreateReplicationController(rc)
				Expect(err).NotTo(HaveOccurred())

				cronJobName := util.ScaledownCronPrefix + restic.Name
				By("Checking cron job created: " + cronJobName)
				Eventually(func() error {
					_, err := f.KubeClient.BatchV1beta1().CronJobs(restic.Namespace).Get(cronJobName, metav1.GetOptions{})
					return err
				}).Should(BeNil())

				By("Waiting for scale replication controller to 0 replica")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveReplica(0))

				By("Waiting for scale up replication controller to 1 replica")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveReplica(1))

				By("Waiting for init-container")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveInitContainer(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(repos.Items).NotTo(BeEmpty())
				f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Waiting for scale up replication controller to original replica")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveReplica(int(*rc.Spec.Replicas)))
			})
		})
	})

	Describe("Pause Restic to stop backup", func() {
		Context(`"Local" backend`, func() {
			AfterEach(func() {
				f.DeleteReplicationController(rc.ObjectMeta)
				f.DeleteRestic(restic.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
			})
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should be able to Pause and Resume backup`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating ReplicationController " + rc.Name)
				_, err = f.CreateReplicationController(rc)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(repos.Items).NotTo(BeEmpty())
				f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By(`Patching Restic with "paused: true"`)
				err = f.CreateOrPatchRestic(restic.ObjectMeta, func(in *api.Restic) *api.Restic {
					in.Spec.Paused = true
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				repos, err = f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(repos.Items).NotTo(BeEmpty())

				previousBackupCount := repos.Items[0].Status.BackupCount

				By("Waiting 2 minutes")
				time.Sleep(2 * time.Minute)

				By("Checking that Backup count has not changed")
				repos, err = f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(repos.Items).NotTo(BeEmpty())
				Expect(repos.Items[0].Status.BackupCount).Should(BeNumerically("==", previousBackupCount))

				By(`Patching Restic with "paused: false"`)
				err = f.CreateOrPatchRestic(restic.ObjectMeta, func(in *api.Restic) *api.Restic {
					in.Spec.Paused = false
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for backup to complete")
				f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">", previousBackupCount)))

				By("Waiting for backup event")
				f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">", previousBackupCount)))

			})

		})
	})

	Describe("Create Repository CRD", func() {
		Context(`"Local" backend`, func() {
			AfterEach(func() {
				f.DeleteReplicationController(rc.ObjectMeta)
				f.DeleteRestic(restic.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
			})
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should create Repository CRD`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating ReplicationController " + rc.Name)
				_, err = f.CreateReplicationController(rc)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(repos.Items).NotTo(BeEmpty())
				f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

			})

		})
	})

	Describe("Complete Recovery", func() {
		Context(`"Local" backend, single fileGroup`, func() {
			AfterEach(func() {
				f.DeleteReplicationController(rc.ObjectMeta)
				f.DeleteRestic(restic.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
				f.DeleteRecovery(recovery.ObjectMeta)
				framework.CleanupMinikubeHostPath()
			})
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				recovery = f.RecoveryForRestic(restic)
			})
			It(`recovered volume should have same data`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating ReplicationController " + rc.Name)
				_, err = f.CreateReplicationController(rc)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.GetRepositories(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas))
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(rc.ObjectMeta, &restic)
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting ReplicationController")
				f.DeleteReplicationController(rc.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// give some time for rc to terminate
				time.Sleep(time.Second * 30)

				recovery.Spec.Workload = api.LocalTypedReference{
					Kind: api.KindReplicationController,
					Name: rc.Name,
				}

				By("Creating recovery " + recovery.Name)
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying rc with recovered volume")
				rc.Spec.Template.Spec.Volumes = []core.Volume{
					{
						Name: framework.TestSourceDataVolumeName,
						VolumeSource: core.VolumeSource{
							HostPath: &core.HostPathVolumeSource{
								Path: framework.TestRecoveredVolumePath,
							},
						},
					},
				}
				_, err = f.CreateReplicationController(rc)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(rc.ObjectMeta, &restic).Should(BeEquivalentTo(previousData))
			})

		})

		Context(`"Local" backend, multiple fileGroup`, func() {
			AfterEach(func() {
				f.DeleteReplicationController(rc.ObjectMeta)
				f.DeleteRestic(restic.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
				f.DeleteRecovery(recovery.ObjectMeta)
				framework.CleanupMinikubeHostPath()
			})
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				restic.Spec.FileGroups = framework.FileGroupsForHostPathVolumeWithMultipleDirectory()
				recovery = f.RecoveryForRestic(restic)
			})
			It(`recovered volume should have same data`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating demo data in hostPath")
				err = framework.CreateDemoDataInHostPath()
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				rc.Spec.Template.Spec.Volumes = framework.HostPathVolumeWithMultipleDirectory()
				By("Creating ReplicationController " + rc.Name)
				_, err = f.CreateReplicationController(rc)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.GetRepositories(api.KindReplicationController, rc.ObjectMeta, int(*rc.Spec.Replicas))
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(rc.ObjectMeta, &restic)
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting ReplicationController")
				f.DeleteReplicationController(rc.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// give some time for rc to terminate
				time.Sleep(time.Second * 30)

				recovery.Spec.Workload = api.LocalTypedReference{
					Kind: api.KindReplicationController,
					Name: rc.Name,
				}

				By("Creating recovery " + recovery.Name)
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying rc with recovered volume")
				rc.Spec.Template.Spec.Volumes = []core.Volume{
					{
						Name: framework.TestSourceDataVolumeName,
						VolumeSource: core.VolumeSource{
							HostPath: &core.HostPathVolumeSource{
								Path: framework.TestRecoveredVolumePath,
							},
						},
					},
				}
				_, err = f.CreateReplicationController(rc)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(rc.ObjectMeta, &restic).Should(BeEquivalentTo(previousData))
			})

		})
	})
})
