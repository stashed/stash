package e2e_test

import (
	"time"

	apps_util "github.com/appscode/kutil/apps/v1beta1"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	"github.com/appscode/stash/test/e2e/framework"
	. "github.com/appscode/stash/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1beta1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("StatefulSet", func() {
	var (
		err      error
		f        *framework.Invocation
		restic   api.Restic
		cred     core.Secret
		svc      core.Service
		ss       apps.StatefulSet
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
		recovery.Spec.Backend.StorageSecretName = cred.Name
		svc = f.HeadlessService()
		ss = f.StatefulSet()
	})

	var (
		shouldBackupNewStatefulSet = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating service " + svc.Name)
			err = f.CreateService(svc)
			Expect(err).NotTo(HaveOccurred())

			By("Creating StatefulSet " + ss.Name)
			_, err = f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldBackupExistingStatefulSet = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating service " + svc.Name)
			err = f.CreateService(svc)
			Expect(err).NotTo(HaveOccurred())

			By("Creating StatefulSet " + ss.Name)
			_, err = f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveSidecar(util.StashContainer))

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

			By("Creating service " + svc.Name)
			err = f.CreateService(svc)
			Expect(err).NotTo(HaveOccurred())

			By("Creating StatefulSet " + ss.Name)
			_, err = f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Deleting restic " + restic.Name)
			f.DeleteRestic(restic.ObjectMeta)

			By("Wating to remove sidecar")
			f.EventuallyStatefulSet(ss.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldStopBackupIfLabelChanged = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating service " + svc.Name)
			err = f.CreateService(svc)
			Expect(err).NotTo(HaveOccurred())

			By("Creating StatefulSet " + ss.Name)
			_, err = f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Removing labels of StatefulSet " + ss.Name)
			_, _, err = apps_util.PatchStatefulSet(f.KubeClient, &ss, func(in *apps.StatefulSet) *apps.StatefulSet {
				in.Labels = map[string]string{
					"app": "unmatched",
				}
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting to remove sidecar")
			f.EventuallyStatefulSet(ss.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldStopBackupIfSelectorChanged = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating service " + svc.Name)
			err = f.CreateService(svc)
			Expect(err).NotTo(HaveOccurred())

			By("Creating StatefulSet " + ss.Name)
			_, err = f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveSidecar(util.StashContainer))

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

			f.EventuallyStatefulSet(ss.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldRestoreStatefulSet = func() {
			shouldBackupNewStatefulSet()
			recovery.Spec.Workload = api.LocalTypedReference{
				Kind: api.KindStatefulSet,
				Name: ss.Name,
			}
			recovery.Spec.PodOrdinal = "0"

			By("Creating recovery " + recovery.Name)
			err = f.CreateRecovery(recovery)
			Expect(err).NotTo(HaveOccurred())

			f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())
		}

		shouldInitializeAndBackupStatefulSet = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating service " + svc.Name)
			err = f.CreateService(svc)
			Expect(err).NotTo(HaveOccurred())

			By("Creating StatefulSet " + ss.Name)
			obj, err := f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())

			// sidecar should be added as soon as workload created, we don't need to wait for it
			By("Checking sidecar created")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *api.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}
	)

	Describe("Creating restic for", func() {
		AfterEach(func() {
			f.DeleteStatefulSet(ss.ObjectMeta)
			f.DeleteService(svc.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should backup new StatefulSet`, shouldBackupNewStatefulSet)
			It(`should backup existing StatefulSet`, shouldBackupExistingStatefulSet)
		})

		Context(`"S3" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForS3Backend()
				restic = f.ResticForS3Backend()
			})
			It(`should backup new StatefulSet`, shouldBackupNewStatefulSet)
			It(`should backup existing StatefulSet`, shouldBackupExistingStatefulSet)
		})

		Context(`"DO" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForDOBackend()
				restic = f.ResticForDOBackend()
			})
			It(`should backup new StatefulSet`, shouldBackupNewStatefulSet)
			It(`should backup existing StatefulSet`, shouldBackupExistingStatefulSet)
		})

		Context(`"GCS" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForGCSBackend()
				restic = f.ResticForGCSBackend()
			})
			It(`should backup new StatefulSet`, shouldBackupNewStatefulSet)
			It(`should backup existing StatefulSet`, shouldBackupExistingStatefulSet)
		})

		Context(`"Azure" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForAzureBackend()
				restic = f.ResticForAzureBackend()
			})
			It(`should backup new StatefulSet`, shouldBackupNewStatefulSet)
			It(`should backup existing StatefulSet`, shouldBackupExistingStatefulSet)
		})

		Context(`"Swift" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForSwiftBackend()
				restic = f.ResticForSwiftBackend()
			})
			It(`should backup new StatefulSet`, shouldBackupNewStatefulSet)
			It(`should backup existing StatefulSet`, shouldBackupExistingStatefulSet)
		})

		Context(`"B2" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForB2Backend()
				restic = f.ResticForB2Backend()
			})
			It(`should backup new StatefulSet`, shouldBackupNewStatefulSet)
			It(`should backup existing StatefulSet`, shouldBackupExistingStatefulSet)
		})
	})

	Describe("Changing StatefulSet labels", func() {
		AfterEach(func() {
			f.DeleteStatefulSet(ss.ObjectMeta)
			f.DeleteService(svc.ObjectMeta)
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
			f.DeleteStatefulSet(ss.ObjectMeta)
			f.DeleteService(svc.ObjectMeta)
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
			f.DeleteStatefulSet(ss.ObjectMeta)
			f.DeleteService(svc.ObjectMeta)
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
			f.DeleteStatefulSet(ss.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
			f.DeleteService(svc.ObjectMeta)
			f.DeleteRecovery(recovery.ObjectMeta)
			framework.CleanupMinikubeHostPath()
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				recovery = f.RecoveryForRestic(restic)
			})
			It(`should restore local StatefulSet backup`, shouldRestoreStatefulSet)
		})

		Context(`"S3" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForS3Backend()
				restic = f.ResticForS3Backend()
				recovery = f.RecoveryForRestic(restic)
			})
			It(`should restore s3 StatefulSet backup`, shouldRestoreStatefulSet)
		})
	})

	Describe("Stash initializer for", func() {
		AfterEach(func() {
			f.DeleteStatefulSet(ss.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteService(svc.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It("should initialize and backup new StatefulSet", shouldInitializeAndBackupStatefulSet)
		})
	})

	Describe("Offline backup for", func() {
		AfterEach(func() {
			f.DeleteStatefulSet(ss.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
			f.DeleteService(svc.ObjectMeta)
			framework.CleanupMinikubeHostPath()
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				restic.Spec.Type = api.BackupOffline
				restic.Spec.Schedule = "*/5 * * * *"
			})
			It(`should backup new StatefulSet`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				cronJobName := util.KubectlCronPrefix + restic.Name
				By("Checking cron job created: " + cronJobName)
				Eventually(func() error {
					_, err := f.KubeClient.BatchV1beta1().CronJobs(restic.Namespace).Get(cronJobName, metav1.GetOptions{})
					return err
				}).Should(BeNil())

				By("Creating service " + svc.Name)
				err = f.CreateService(svc)
				Expect(err).NotTo(HaveOccurred())

				By("Creating StatefulSet " + ss.Name)
				_, err = f.CreateStatefulSet(ss)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for init-container")
				f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveInitContainer(util.StashContainer))

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
			})
		})
	})

	Describe("Pause Restic to stop backup", func() {
		Context(`"Local" backend`, func() {
			AfterEach(func() {
				f.DeleteStatefulSet(ss.ObjectMeta)
				f.DeleteService(svc.ObjectMeta)
				f.DeleteRestic(restic.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
			})
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should able to Pause and Resume backup`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating service " + svc.Name)
				err = f.CreateService(svc)
				Expect(err).NotTo(HaveOccurred())

				By("Creating StatefulSet " + ss.Name)
				_, err = f.CreateStatefulSet(ss)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveSidecar(util.StashContainer))

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
