package e2e_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/go/types"
	"github.com/appscode/stash/apis"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	"github.com/appscode/stash/test/e2e/framework"
	. "github.com/appscode/stash/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apps_util "kmodules.xyz/client-go/apps/v1"
	core_util "kmodules.xyz/client-go/core/v1"
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
		localRef     api.LocalTypedReference
	)

	BeforeEach(func() {
		f = root.Invoke()
	})
	AfterEach(func() {
		err := framework.WaitUntilDeploymentDeleted(f.KubeClient, deployment.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, deployment.Namespace, f.AppLabel())
		Expect(err).NotTo(HaveOccurred())

		err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		f.DeleteRepositories(f.DeploymentRepos(&deployment))

		err = framework.WaitUntilRepositoriesDeleted(f.StashClient, f.DeploymentRepos(&deployment))
		Expect(err).NotTo(HaveOccurred())
	})
	JustBeforeEach(func() {
		if missing, _ := BeZero().Match(cred); missing {
			Skip("Missing repository credential")
		}
		restic.Spec.Backend.StorageSecretName = cred.Name
		secondRestic.Spec.Backend.StorageSecretName = cred.Name
		pvc := f.GetPersistentVolumeClaim()
		err := f.CreatePersistentVolumeClaim(pvc)
		Expect(err).NotTo(HaveOccurred())
		deployment = f.Deployment(pvc.Name)
		localRef = api.LocalTypedReference{
			Kind: apis.KindDeployment,
			Name: deployment.Name,
		}

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
			f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos := f.DeploymentRepos(&deployment)
			Expect(repos).NotTo(BeEmpty())
			f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
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
			f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos := f.DeploymentRepos(&deployment)
			Expect(repos).NotTo(BeEmpty())
			f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
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
			f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

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
			f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Removing labels of Deployment " + deployment.Name)
			_, _, err = apps_util.PatchDeployment(f.KubeClient, &deployment, func(in *apps.Deployment) *apps.Deployment {
				in.Labels = map[string]string{
					"app": "unmatched",
				}
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting to remove sidecar")
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
			f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

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

			By("Waiting to remove sidecar")
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
			f.CheckLeaderElection(deployment.ObjectMeta, apis.KindDeployment, api.ResourceKindRestic)

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos := f.DeploymentRepos(&deployment)
			Expect(repos).NotTo(BeEmpty())
			f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
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
			f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos := f.DeploymentRepos(&deployment)
			Expect(repos).NotTo(BeEmpty())
			f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
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
			f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

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
			f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

			By("Waiting for backup to complete")
			f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
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
			err := framework.WaitUntilRecoveryDeleted(f.StashClient, recovery.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				recovery = f.RecoveryForRestic(restic)
			})
			It(`should delete job after recovery deleted`, func() {

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
				f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

				repos := f.DeploymentRepos(&deployment)
				Expect(repos).NotTo(BeEmpty())

				recovery.Spec.Repository.Name = repos[0].Name
				recovery.Spec.Repository.Namespace = f.Namespace()
				By("Creating recovery " + recovery.Name)
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				jobName := util.RecoveryJobPrefix + recovery.Name

				By("Checking Job exists")
				Eventually(func() bool {
					_, err := f.KubeClient.BatchV1().Jobs(recovery.Namespace).Get(jobName, metav1.GetOptions{})
					return err == nil
				}, time.Minute*3, time.Second*2).Should(BeTrue())

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
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteRestic(secondRestic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)

			err := framework.WaitUntilResticDeleted(f.StashClient, secondRestic.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
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
		})

		Context(`Single Replica`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				restic.Spec.Type = api.BackupOffline
				restic.Spec.Schedule = "@every 3m"
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

				start := time.Now()
				for i := 1; i <= 3; i++ {
					fmt.Printf("=============== Waiting for backup no: %d =============\n", i)
					By("Waiting for scale down deployment to 0 replica")
					f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveReplica(0))

					By("Waiting for scale up deployment to 1 replica")
					f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveReplica(1))

					By("Waiting for init-container")
					f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveInitContainer(util.StashContainer))

					By("Waiting for Repository CRD")
					f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

					By("Waiting for backup to complete")
					f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically("==", i)))
				}
				elapsedTime := time.Since(start).Minutes()

				// backup is scheduled for every 3 minutes.
				// so 3 backup by cronJob should not take more than 9 minutes + some overhead.(let 1 minute overhead for each backup)
				Expect(elapsedTime).Should(BeNumerically("<=", 9+3))

				By("Waiting for backup event")
				repos := f.DeploymentRepos(&deployment)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Waiting for scale up deployment to original replica")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveReplica(int(*deployment.Spec.Replicas)))
			})
		})

		Context("Multiple Replica", func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				restic.Spec.Type = api.BackupOffline
				restic.Spec.Schedule = "@every 3m"
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

				start := time.Now()
				for i := 1; i <= 3; i++ {
					fmt.Printf("=============== Waiting for backup no: %d =============\n", i)
					By("Waiting for scale down deployment to 0 replica")
					f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveReplica(0))

					By("Waiting for scale up deployment to 1 replica")
					f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveReplica(1))

					By("Waiting for init-container")
					f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveInitContainer(util.StashContainer))

					By("Waiting for Repository CRD")
					f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

					By("Waiting for backup to complete")
					f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", i)))
				}
				elapsedTime := time.Since(start).Minutes()

				// backup is scheduled for every 3 minutes.
				// so 3 backup by cronJob should not take more than 9 minutes + some overhead.(let 1 minute overhead for each backup)
				Expect(elapsedTime).Should(BeNumerically("<=", 9+3))

				By("Waiting for backup event")
				repos := f.DeploymentRepos(&deployment)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

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
				addrs, err := f.CreateMinioServer(true, nil)
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
				f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.DeploymentRepos(&deployment)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

			})
		})
		Context("Without cacert", func() {
			BeforeEach(func() {
				By("Creating Minio server with cacert")
				addrs, err := f.CreateMinioServer(true, nil)
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
				f.EventualWarning(restic.ObjectMeta, framework.KindRestic).Should(WithTransform(f.CountFailedSetup, BeNumerically(">=", 1)))

				By("Checking Repository CRD not created")
				repos := f.DeploymentRepos(&deployment)
				Expect(repos).To(BeEmpty())
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

			err := framework.WaitUntilSecretDeleted(f.KubeClient, registryCred.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
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
				f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.DeploymentRepos(&deployment)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By(`Patching Restic with "paused: true"`)
				err = f.CreateOrPatchRestic(restic.ObjectMeta, func(in *api.Restic) *api.Restic {
					in.Spec.Paused = true
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				// wait some time for ongoing backup
				time.Sleep(time.Second * 30)
				repos = f.DeploymentRepos(&deployment)

				Expect(repos).NotTo(BeEmpty())

				previousBackupCount := repos[0].Status.BackupCount

				By("Waiting 2 minutes")
				time.Sleep(2 * time.Minute)

				By("Checking that Backup count has not changed")
				repos = f.DeploymentRepos(&deployment)
				Expect(err).NotTo(HaveOccurred())
				Expect(repos).NotTo(BeEmpty())
				Expect(repos[0].Status.BackupCount).Should(BeNumerically("==", previousBackupCount))

				By(`Patching Restic with "paused: false"`)
				err = f.CreateOrPatchRestic(restic.ObjectMeta, func(in *api.Restic) *api.Restic {
					in.Spec.Paused = false
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">", previousBackupCount)))

				By("Waiting for backup event")
				repos = f.DeploymentRepos(&deployment)
				Expect(err).NotTo(HaveOccurred())
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">", previousBackupCount)))

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
				f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.DeploymentRepos(&deployment)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

			})

		})
	})

	Describe("Complete Recovery", func() {
		AfterEach(func() {
			f.CleanupRecoveredVolume(deployment.ObjectMeta)
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
			f.DeleteRecovery(recovery.ObjectMeta)

			err := framework.WaitUntilRecoveryDeleted(f.StashClient, recovery.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})
		Context(`"Local" backend,single fileGroup`, func() {
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
				f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.DeploymentRepos(&deployment)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(deployment.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic))
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting deployment")
				f.DeleteDeployment(deployment.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// wait until deployment terminated
				err = framework.WaitUntilDeploymentDeleted(f.KubeClient, deployment.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, deployment.Namespace, f.AppLabel())
				Expect(err).NotTo(HaveOccurred())

				recovery.Spec.Repository.Name = localRef.GetRepositoryCRDName("", "")
				recovery.Spec.Repository.Namespace = f.Namespace()

				By("Creating recovery " + recovery.Name)
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying deployment with recovered volume")
				deployment.Spec.Template.Spec.Volumes = f.RecoveredVolume()
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(deployment.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic)).Should(BeEquivalentTo(previousData))
			})

		})

		Context(`"Local" backend, multiple fileGroup`, func() {
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

				deployment.Spec.Template.Spec.Volumes = f.HostPathVolumeWithMultipleDirectory()
				By("Creating Deployment " + deployment.Name)
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())
				apps_util.WaitUntilDeploymentReady(f.KubeClient, deployment.ObjectMeta)

				By("Creating demo data in hostPath")
				err = f.CreateDemoData(deployment.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.DeploymentRepos(&deployment)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(deployment.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic))
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting deployment")
				f.DeleteDeployment(deployment.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// wait until deployment terminated
				err = framework.WaitUntilDeploymentDeleted(f.KubeClient, deployment.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, deployment.Namespace, f.AppLabel())
				Expect(err).NotTo(HaveOccurred())

				recovery.Spec.Repository.Name = localRef.GetRepositoryCRDName("", "")
				recovery.Spec.Repository.Namespace = f.Namespace()

				By("Creating recovery " + recovery.Name)
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying deployment with recovered volume")
				deployment.Spec.Template.Spec.Volumes = f.RecoveredVolume()
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(deployment.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic)).Should(BeEquivalentTo(previousData))
			})

		})
	})

	Describe("Recover from different namespace", func() {
		var (
			recoveryNamespace *core.Namespace
		)
		AfterEach(func() {
			f.CleanupRecoveredVolume(deployment.ObjectMeta)
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
			f.DeleteRecovery(recovery.ObjectMeta)

			err := framework.WaitUntilRecoveryDeleted(f.StashClient, recovery.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})

		Context(`"Local" backend,single fileGroup`, func() {
			AfterEach(func() {
				f.DeleteNamespace(recoveryNamespace.Name)

				err := framework.WaitUntilNamespaceDeleted(f.KubeClient, recoveryNamespace.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
			})
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				recovery = f.RecoveryForRestic(restic)
				recoveryNamespace = f.NewNamespace(rand.WithUniqSuffix("recovery"))
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
				f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.DeploymentRepos(&deployment)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(deployment.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic))
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting deployment")
				f.DeleteDeployment(deployment.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// wait until deployment terminated
				err = framework.WaitUntilDeploymentDeleted(f.KubeClient, deployment.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, deployment.Namespace, f.AppLabel())
				Expect(err).NotTo(HaveOccurred())

				By("Creating new namespace: " + recoveryNamespace.Name)
				err = f.CreateNamespace(recoveryNamespace)
				Expect(err).NotTo(HaveOccurred())

				recovery.Spec.Repository.Name = localRef.GetRepositoryCRDName("", "")
				recovery.Spec.Repository.Namespace = f.Namespace()

				By("Creating recovery " + recovery.Name)
				recovery.Namespace = recoveryNamespace.Name
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying deployment with recovered volume")
				deployment.Namespace = recoveryNamespace.Name
				deployment.Spec.Template.Spec.Volumes = f.RecoveredVolume()
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(deployment.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic)).Should(BeEquivalentTo(previousData))
			})

		})
	})

	Describe("Recover specific snapshot", func() {

		Context(`"Local" backend, multiple fileGroup`, func() {
			AfterEach(func() {
				f.CleanupRecoveredVolume(deployment.ObjectMeta)
				f.DeleteDeployment(deployment.ObjectMeta)
				f.DeleteRestic(restic.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
				f.DeleteRecovery(recovery.ObjectMeta)

				err := framework.WaitUntilRecoveryDeleted(f.StashClient, recovery.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
			})
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				restic.Spec.FileGroups = framework.FileGroupsForHostPathVolumeWithMultipleDirectory()
				recovery = f.RecoveryForRestic(restic)
			})
			It(`recovered volume should have old data`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				deployment.Spec.Template.Spec.Volumes = f.HostPathVolumeWithMultipleDirectory()
				By("Creating Deployment " + deployment.Name)
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())
				apps_util.WaitUntilDeploymentReady(f.KubeClient, deployment.ObjectMeta)

				By("Creating demo data in hostPath")
				err = f.CreateDemoData(deployment.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.DeploymentRepos(&deployment)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				repos = f.DeploymentRepos(&deployment)
				Expect(repos).NotTo(BeEmpty())
				previousBackupCount := repos[0].Status.BackupCount

				By("Listing old snapshots")
				oldSnapshots, err := f.StashClient.RepositoriesV1alpha1().Snapshots(f.Namespace()).List(metav1.ListOptions{LabelSelector: "repository=" + repos[0].Name})
				Expect(err).NotTo(HaveOccurred())

				latestOldSnapashot := f.LatestSnapshot(oldSnapshots.Items) // latest snapshot of oldSnapshots

				By("Reading data from mountPath")
				oldData, err := f.ReadDataFromMountedDir(deployment.ObjectMeta, latestOldSnapashot.Status.Paths)
				Expect(err).NotTo(HaveOccurred())
				Expect(oldData).NotTo(BeEmpty())

				By("Creating new data on mountPath")
				err = f.CreateDataOnMountedDir(deployment.ObjectMeta, latestOldSnapashot.Status.Paths, "test-data.txt")
				Expect(err).NotTo(HaveOccurred())

				By("Reading new data from mountPath")
				newData, err := f.ReadDataFromMountedDir(deployment.ObjectMeta, latestOldSnapashot.Status.Paths)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for new backup")
				f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">", previousBackupCount)))

				By("Deleting deployment")
				f.DeleteDeployment(deployment.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// wait until deployment terminated
				err = framework.WaitUntilDeploymentDeleted(f.KubeClient, deployment.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, deployment.Namespace, f.AppLabel())
				Expect(err).NotTo(HaveOccurred())

				recovery.Spec.Repository.Name = localRef.GetRepositoryCRDName("", "")
				recovery.Spec.Repository.Namespace = f.Namespace()

				By("Creating recovery " + recovery.Name)
				recovery.Spec.Snapshot = latestOldSnapashot.Name
				recovery.Spec.Paths = latestOldSnapashot.Status.Paths
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying deployment with recovered volume")
				deployment.Spec.Template.Spec.Volumes = f.RecoveredVolume()
				_, err = f.CreateDeployment(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Checking data recovered from old snapshot")
				f.EventuallyRecoveredData(deployment.ObjectMeta, latestOldSnapashot.Status.Paths).ShouldNot(BeEquivalentTo(newData))
				f.EventuallyRecoveredData(deployment.ObjectMeta, latestOldSnapashot.Status.Paths).Should(BeEquivalentTo(oldData))
			})

		})
	})

	Describe("Repository WipeOut", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context(`"Minio" backend`, func() {
			AfterEach(func() {
				f.DeleteMinioServer()
			})
			BeforeEach(func() {
				clusterIP := net.IP{192, 168, 99, 100}

				pod, err := f.GetOperatorPod()
				if pod.Spec.NodeName != "minikube" {
					node, err := f.KubeClient.CoreV1().Nodes().Get(pod.Spec.NodeName, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					for _, addr := range node.Status.Addresses {
						if addr.Type == core.NodeExternalIP {
							clusterIP = net.ParseIP(addr.Address)
							break
						}
					}
				}

				By("Creating Minio server with cacert")
				_, err = f.CreateMinioServer(true, []net.IP{clusterIP})
				Expect(err).NotTo(HaveOccurred())

				msvc, err := f.KubeClient.CoreV1().Services(f.Namespace()).Get("minio-service", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				minioServiceNodePort := strconv.Itoa(int(msvc.Spec.Ports[0].NodePort))

				restic = f.ResticForMinioBackend("https://" + clusterIP.String() + ":" + minioServiceNodePort)
				cred = f.SecretForMinioBackend(true)
			})
			It(`should delete repository from minio backend`, func() {
				shouldBackupNewDeployment()

				repos := f.DeploymentRepos(&deployment)
				Expect(repos).NotTo(BeEmpty())

				By(`Patching Repository with "wipeOut: true"`)
				err = f.CreateOrPatchRepository(repos[0].ObjectMeta, func(in *api.Repository) *api.Repository {
					in.Spec.WipeOut = true
					return in
				})

				By("Deleting Repository CRD")
				f.DeleteRepositories(repos)

				By("Waiting for repository to delete")
				f.EventuallyRepository(&deployment).Should(BeEmpty())

				By("Checking restic repository is deleted")
				items, err := f.BrowseResticRepository(repos[0])
				Expect(err).ShouldNot(HaveOccurred())
				Expect(items).Should(BeEmpty())
			})

		})

		Context(`"Local" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should reject to to patch repository with wipeOut: true`, func() {
				shouldBackupNewDeployment()

				repos := f.DeploymentRepos(&deployment)
				Expect(repos).NotTo(BeEmpty())

				By(`Patching Repository with "wipeOut: true"`)
				err = f.CreateOrPatchRepository(repos[0].ObjectMeta, func(in *api.Repository) *api.Repository {
					in.Spec.WipeOut = true
					return in
				})
				Expect(err).Should(HaveOccurred())
			})
		})

	})

	Describe("CheckJob", func() {
		AfterEach(func() {
			f.DeleteDeployment(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context("Multiple Replica", func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				restic.Spec.Type = api.BackupOffline
				restic.Spec.Schedule = "*/3 * * * *"
			})
			It(`should create checkJob`, func() {
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
				f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.DeploymentRepos(&deployment)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Waiting for scale up deployment to original replica")
				f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveReplica(int(*deployment.Spec.Replicas)))

				By("Checking checkjob created")
				checkJobName := util.CheckJobPrefix + restic.Name
				f.EventuallyJobSucceed(checkJobName).Should(BeTrue())

			})
		})
	})
})
