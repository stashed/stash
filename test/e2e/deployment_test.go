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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Deployment", func() {
	var (
		err          error
		f            *framework.Invocation
		restic       api.Restic
		secondRestic api.Restic
		cred         core.Secret
		deployment   apps.Deployment
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

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(repos.Items).NotTo(BeEmpty())
			f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
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

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

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

			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Deleting restic " + restic.Name)
			f.DeleteRestic(restic.ObjectMeta)

			By("Waiting to remove sidecar")
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

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

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

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

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

			By("Waiting for leader election")
			f.CheckLeaderElection(deployment.ObjectMeta, api.KindDeployment)

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(repos.Items).NotTo(BeEmpty())
			f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldMutateAndBackupNewDeployment = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			obj, err := f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			// sidecar should be added as soon as deployment created, we don't need to wait for it
			By("Checking sidecar created")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(repos.Items).NotTo(BeEmpty())
			f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldNotMutateNewDeployment = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			obj, err := f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar not added")
			Expect(obj).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldRejectToCreateNewDeployment = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating first restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating second restic " + secondRestic.Name)
			err = f.CreateRestic(secondRestic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			_, err := f.CreateDeployment(deployment)
			Expect(err).To(HaveOccurred())
		}

		shouldRemoveSidecarInstantly = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			obj, err := f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar added")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Removing labels of Deployment " + deployment.Name)
			obj, _, err = apps_util.PatchDeployment(f.KubeClient, &deployment, func(in *apps.Deployment) *apps.Deployment {
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

			By("Creating Deployment " + deployment.Name)
			previousLabel := deployment.Labels
			deployment.Labels = map[string]string{
				"app": "unmatched",
			}
			obj, err := f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar not added")
			Expect(obj).ShouldNot(HaveSidecar(util.StashContainer))

			By("Adding label to match restic" + deployment.Name)
			obj, _, err = apps_util.PatchDeployment(f.KubeClient, &deployment, func(in *apps.Deployment) *apps.Deployment {
				in.Labels = previousLabel
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar added")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
		}
	)

	Describe("Creating restic for", func() {
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
			It(`should backup new Deployment`, shouldBackupNewDeployment)
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

		Context(`"B2" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForB2Backend()
				restic = f.ResticForB2Backend()
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

		Context(`"B2" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForB2Backend()
				restic = f.ResticForB2Backend()
			})
			It(`should stop backup`, shouldStopBackup)
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
				recovery = f.RecoveryForRestic(restic)
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
				f.DeleteJobAndDependents(jobName, &recovery)
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

	Describe("Stash Webhook for", func() {
		BeforeEach(func() {
			if !f.WebhookEnabled {
				Skip("Webhook is disabled")
			}
		})
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteRestic(secondRestic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
				secondRestic = restic
				secondRestic.Name = "second-restic"
			})
			It("should mutate and backup new Deployment", shouldMutateAndBackupNewDeployment)
			It("should not mutate new Deployment if no restic select it", shouldNotMutateNewDeployment)
			It("should reject to create new Deployment if multiple restic select it", shouldRejectToCreateNewDeployment)
			It("should remove sidecar instantly if label change to match no restic", shouldRemoveSidecarInstantly)
			It("should add sidecar instantly if label change to match single restic", shouldAddSidecarInstantly)
		})
	})

	Describe("Offline backup for", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
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
			It(`should backup new Deployment`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating Deployment " + deployment.Name)
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				cronJobName := util.ScaledownCronPrefix + restic.Name
				By("Checking cron job created: " + cronJobName)
				Eventually(func() error {
					_, err := f.KubeClient.BatchV1beta1().CronJobs(restic.Namespace).Get(cronJobName, metav1.GetOptions{})
					return err
				}).Should(BeNil())

				By("Waiting for scale down deployment to 0 replica")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveReplica(0))

				By("Waiting for scale up deployment to 1 replica")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveReplica(1))

				By("Waiting for init-container")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveInitContainer(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(repos.Items).NotTo(BeEmpty())
				f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Waiting for scale up deployment to original replica")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveReplica(int(*deployment.Spec.Replicas)))
			})
		})

		Context("Multiple Replica", func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				restic.Spec.Type = api.BackupOffline
				restic.Spec.Schedule = "*/5 * * * *"
			})
			It(`should backup new Deployment`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating Deployment " + deployment.Name)
				deployment.Spec.Replicas = types.Int32P(3)
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				cronJobName := util.ScaledownCronPrefix + restic.Name
				By("Checking cron job created: " + cronJobName)
				Eventually(func() error {
					_, err := f.KubeClient.BatchV1beta1().CronJobs(restic.Namespace).Get(cronJobName, metav1.GetOptions{})
					return err
				}).Should(BeNil())

				By("Waiting for scale down deployment to 0 replica")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveReplica(0))

				By("Waiting for scale up deployment to 1 replica")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveReplica(1))

				By("Waiting for init-container")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveInitContainer(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(repos.Items).NotTo(BeEmpty())
				f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Waiting for scale up deployment to original replica")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveReplica(int(*deployment.Spec.Replicas)))
			})
		})
	})

	Describe("No retention policy", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
				restic.Spec.FileGroups[0].RetentionPolicyName = ""
				restic.Spec.RetentionPolicies = []api.RetentionPolicy{}
			})
			It(`should backup new Deployment`, shouldBackupNewDeployment)
		})
	})
	Describe("Minio server", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
			f.DeleteMinioServer()
		})
		Context("With cacert", func() {
			BeforeEach(func() {
				By("Creating Minio server with cacert")
				addrs, err := f.CreateMinioServer()
				Expect(err).NotTo(HaveOccurred())

				restic = f.ResticForMinioBackend("https://" + addrs)
				cred = f.SecretForMinioBackend(true)

			})

			It("Should backup new Deployment", func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating Deployment " + deployment.Name)
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(repos.Items).NotTo(BeEmpty())
				f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

			})
		})
		Context("Without cacert", func() {
			BeforeEach(func() {
				By("Creating Minio server with cacert")
				addrs, err := f.CreateMinioServer()
				Expect(err).NotTo(HaveOccurred())

				restic = f.ResticForMinioBackend("https://" + addrs)
				cred = f.SecretForMinioBackend(false)

			})

			It("Should fail to backup new Deployment", func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic without cacert")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating Deployment " + deployment.Name)
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting to count failed setup event")
				f.EventualWarning(restic.ObjectMeta).Should(WithTransform(f.CountFailedSetup, BeNumerically(">=", 1)))

				By("Waiting to count successful backup event")
				repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(repos.Items).NotTo(BeEmpty())
				f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically("==", 0)))

			})
		})
	})

	Describe("Private docker registry", func() {
		var registryCred core.Secret
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
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
			restic = f.ResticForLocalBackend()
			restic.Spec.ImagePullSecrets = []core.LocalObjectReference{
				{
					Name: registryCred.Name,
				},
			}
		})
		It(`should backup new Deployment`, shouldBackupNewDeployment)
	})

	Describe("Pause Restic to stop backup", func() {
		Context(`"Local" backend`, func() {
			AfterEach(func() {
				f.DeleteDeployment(deployment.ObjectMeta)
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

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating Deployment " + deployment.Name)
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

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
				f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">", previousBackupCount)))

				By("Waiting for backup event")
				repos, err = f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(repos.Items).NotTo(BeEmpty())
				f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">", previousBackupCount)))

			})

		})
	})

	Describe("Repository CRD", func() {
		Context(`"Local" backend`, func() {
			AfterEach(func() {
				f.DeleteDeployment(deployment.ObjectMeta)
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

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating Deployment " + deployment.Name)
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(repos.Items).NotTo(BeEmpty())
				f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

			})

		})
	})

	Describe("Complete Recovery", func() {
		Context(`"Local" backend,single fileGroup`, func() {
			AfterEach(func() {
				f.DeleteDeployment(deployment.ObjectMeta)
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

				By("Creating Deployment " + deployment.Name)
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.GetRepositories(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas))
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(deployment.ObjectMeta, &restic)
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting deployment")
				f.DeleteDeployment(deployment.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// give some time for deployment to terminate
				time.Sleep(time.Second * 30)

				recovery.Spec.Workload = api.LocalTypedReference{
					Kind: api.KindDeployment,
					Name: deployment.Name,
				}

				By("Creating recovery " + recovery.Name)
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying deployment with recovered volume")
				deployment.Spec.Template.Spec.Volumes = []core.Volume{
					{
						Name: framework.TestSourceDataVolumeName,
						VolumeSource: core.VolumeSource{
							HostPath: &core.HostPathVolumeSource{
								Path: framework.TestRecoveredVolumePath,
							},
						},
					},
				}
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(deployment.ObjectMeta, &restic).Should(BeEquivalentTo(previousData))
			})

		})

		Context(`"Local" backend, multiple fileGroup`, func() {
			AfterEach(func() {
				f.DeleteDeployment(deployment.ObjectMeta)
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

				deployment.Spec.Template.Spec.Volumes = framework.HostPathVolumeWithMultipleDirectory()
				By("Creating Deployment " + deployment.Name)
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas)).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.GetRepositories(api.KindDeployment, deployment.ObjectMeta, int(*deployment.Spec.Replicas))
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(deployment.ObjectMeta, &restic)
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting deployment")
				f.DeleteDeployment(deployment.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// give some time for deployment to terminate
				time.Sleep(time.Second * 30)

				recovery.Spec.Workload = api.LocalTypedReference{
					Kind: api.KindDeployment,
					Name: deployment.Name,
				}

				By("Creating recovery " + recovery.Name)
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying deployment with recovered volume")
				deployment.Spec.Template.Spec.Volumes = []core.Volume{
					{
						Name: framework.TestSourceDataVolumeName,
						VolumeSource: core.VolumeSource{
							HostPath: &core.HostPathVolumeSource{
								Path: framework.TestRecoveredVolumePath,
							},
						},
					},
				}
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(deployment.ObjectMeta, &restic).Should(BeEquivalentTo(previousData))
			})

		})
	})
})
