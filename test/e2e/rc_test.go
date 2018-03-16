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
		err      error
		f        *framework.Invocation
		restic   api.Restic
		restic2  api.Restic
		cred     core.Secret
		rc       core.ReplicationController
		recovery api.Recovery
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
		restic.Spec.Backend.StorageSecretName = cred.Name
		restic2.Spec.Backend.StorageSecretName = cred.Name
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

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
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

			By("Creating ReplicationController " + rc.Name)
			_, err = f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Deleting restic " + restic.Name)
			f.DeleteRestic(restic.ObjectMeta)

			By("Wating to remove sidecar")
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

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

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

			f.EventuallyReplicationController(rc.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldRestoreRC = func() {
			shouldBackupNewReplicationController()
			recovery.Spec.Workload = api.LocalTypedReference{
				Kind: api.KindReplicationController,
				Name: rc.Name,
			}

			By("Creating recovery " + recovery.Name)
			err = f.CreateRecovery(recovery)
			Expect(err).NotTo(HaveOccurred())

			f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())
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

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
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

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
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

			By("Creating second restic " + restic2.Name)
			err = f.CreateRestic(restic2)
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

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

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

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))
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

	Describe("Creating recovery for", func() {
		AfterEach(func() {
			f.DeleteReplicationController(rc.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
			f.DeleteRecovery(recovery.ObjectMeta)
			framework.CleanupMinikubeHostPath()
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				recovery = f.RecoveryForRestic(restic)
			})
			It(`should restore local rc backup`, shouldRestoreRC)
		})

		Context(`"S3" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForS3Backend()
				restic = f.ResticForS3Backend()
				recovery = f.RecoveryForRestic(restic)
			})
			It(`should restore s3 rc backup`, shouldRestoreRC)
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
			f.DeleteRestic(restic2.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
				restic2 = restic
				restic2.Name = "restic2"
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

				By("Wating for scale up replication controller to 1 replica")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveReplica(1))

				By("Waiting for init-container")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveInitContainer(util.StashContainer))

				By("Waiting for backup to complete")
				f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
					return r.Status.BackupCount
				}, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

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

				By("Wating for scale up replication controller to 1 replica")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveReplica(1))

				By("Waiting for init-container")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveInitContainer(util.StashContainer))

				By("Waiting for backup to complete")
				f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
					return r.Status.BackupCount
				}, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

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

				By("Waiting for backup to complete")
				f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
					return r.Status.BackupCount
				}, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By(`Patching Restic with "paused: true"`)
				err = f.CreateOrPatchRestic(restic.ObjectMeta, func(in *api.Restic) *api.Restic {
					in.Spec.Paused = true
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				resticObj, err := f.StashClient.StashV1alpha1().Restics(restic.Namespace).Get(restic.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				previousBackupCount := resticObj.Status.BackupCount

				By("Wating 2 minutes")
				time.Sleep(2 * time.Minute)

				By("Checking that Backup count has not changed")
				resticObj, err = f.StashClient.StashV1alpha1().Restics(restic.Namespace).Get(restic.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resticObj.Status.BackupCount).Should(BeNumerically("==", previousBackupCount))

				By(`Patching Restic with "paused: false"`)
				err = f.CreateOrPatchRestic(restic.ObjectMeta, func(in *api.Restic) *api.Restic {
					in.Spec.Paused = false
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for backup to complete")
				f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
					return r.Status.BackupCount
				}, BeNumerically(">", previousBackupCount)))

				By("Waiting for backup event")
				f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">", previousBackupCount)))

			})

		})
	})
})
