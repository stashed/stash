package e2e_test

import (
	"fmt"
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

var _ = Describe("StatefulSet", func() {
	var (
		err          error
		f            *framework.Invocation
		restic       api.Restic
		secondRestic api.Restic
		cred         core.Secret
		svc          core.Service
		ss           apps.StatefulSet
		recovery     api.Recovery
		localRef     api.LocalTypedReference
	)

	BeforeEach(func() {
		f = root.Invoke()
	})
	AfterEach(func() {
		err := framework.WaitUntilStatefulSetDeleted(f.KubeClient, ss.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, ss.Namespace, f.AppLabel())
		Expect(err).NotTo(HaveOccurred())

		err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		err = framework.WaitUntilServiceDeleted(f.KubeClient, svc.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		f.DeleteRepositories(f.StatefulSetRepos(&ss))

		err = framework.WaitUntilRepositoriesDeleted(f.StashClient, f.StatefulSetRepos(&ss))
		Expect(err).NotTo(HaveOccurred())
	})
	JustBeforeEach(func() {
		if missing, _ := BeZero().Match(cred); missing {
			Skip("Missing repository credential")
		}
		restic.Spec.Backend.StorageSecretName = cred.Name
		secondRestic.Spec.Backend.StorageSecretName = cred.Name
		svc = f.HeadlessService()
		pvc := f.GetPersistentVolumeClaim()

		err := f.CreatePersistentVolumeClaim(pvc)
		Expect(err).NotTo(HaveOccurred())
		ss = f.StatefulSet(pvc.Name)
		localRef = api.LocalTypedReference{
			Kind: apis.KindStatefulSet,
			Name: ss.Name,
		}
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

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&ss).Should(WithTransform(func(repoList []*api.Repository) int {
				return len(repoList)
			}, BeNumerically("==", int(*ss.Spec.Replicas))))

			By("Waiting for backup to complete")
			f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos := f.StatefulSetRepos(&ss)
			Expect(repos).NotTo(BeEmpty())
			f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
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

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&ss).Should(WithTransform(func(repoList []*api.Repository) int {
				return len(repoList)
			}, BeNumerically("==", int(*ss.Spec.Replicas))))

			By("Waiting for backup to complete")
			f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos := f.StatefulSetRepos(&ss)
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

			By("Creating service " + svc.Name)
			err = f.CreateService(svc)
			Expect(err).NotTo(HaveOccurred())

			By("Creating StatefulSet " + ss.Name)
			_, err = f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&ss).Should(WithTransform(func(repoList []*api.Repository) int {
				return len(repoList)
			}, BeNumerically("==", int(*ss.Spec.Replicas))))

			By("Waiting for backup to complete")
			f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Deleting restic " + restic.Name)
			f.DeleteRestic(restic.ObjectMeta)

			By("Waiting to remove sidecar")
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

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&ss).Should(WithTransform(func(repoList []*api.Repository) int {
				return len(repoList)
			}, BeNumerically("==", int(*ss.Spec.Replicas))))

			By("Waiting for backup to complete")
			f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

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

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&ss).Should(WithTransform(func(repoList []*api.Repository) int {
				return len(repoList)
			}, BeNumerically("==", int(*ss.Spec.Replicas))))

			By("Waiting for backup to complete")
			f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

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

		shouldMutateAndBackupNewStatefulSet = func() {
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

			// sidecar should be added as soon as ss created, we don't need to wait for it
			By("Checking sidecar created")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&ss).Should(WithTransform(func(repoList []*api.Repository) int {
				return len(repoList)
			}, BeNumerically("==", int(*ss.Spec.Replicas))))

			By("Waiting for backup to complete")
			f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Waiting for backup event")
			repos := f.StatefulSetRepos(&ss)
			Expect(repos).NotTo(BeEmpty())
			f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
		}

		shouldNotMutateNewStatefulSet = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating service " + svc.Name)
			err = f.CreateService(svc)
			Expect(err).NotTo(HaveOccurred())

			By("Creating StatefulSet " + ss.Name)
			obj, err := f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar not added")
			Expect(obj).ShouldNot(HaveSidecar(util.StashContainer))
		}

		shouldRejectToCreateNewStatefulSet = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating first restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating second restic " + secondRestic.Name)
			err = f.CreateRestic(secondRestic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating service " + svc.Name)
			err = f.CreateService(svc)
			Expect(err).NotTo(HaveOccurred())

			By("Creating StatefulSet " + ss.Name)
			_, err := f.CreateStatefulSet(ss)
			Expect(err).To(HaveOccurred())
		}

		shouldRemoveSidecarInstantly = func() {
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

			By("Checking sidecar added")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&ss).Should(WithTransform(func(repoList []*api.Repository) int {
				return len(repoList)
			}, BeNumerically("==", int(*ss.Spec.Replicas))))

			By("Waiting for backup to complete")
			f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Removing labels of StatefulSet " + ss.Name)
			obj, _, err = apps_util.PatchStatefulSet(f.KubeClient, &ss, func(in *apps.StatefulSet) *apps.StatefulSet {
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

			By("Creating service " + svc.Name)
			err = f.CreateService(svc)
			Expect(err).NotTo(HaveOccurred())

			By("Creating StatefulSet " + ss.Name)
			previousLabel := ss.Labels
			ss.Labels = map[string]string{
				"app": "unmatched",
			}
			obj, err := f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar not added")
			Expect(obj).ShouldNot(HaveSidecar(util.StashContainer))

			By("Adding label to match restic" + ss.Name)
			obj, _, err = apps_util.PatchStatefulSet(f.KubeClient, &ss, func(in *apps.StatefulSet) *apps.StatefulSet {
				in.Labels = previousLabel
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking sidecar added")
			Expect(obj).Should(HaveSidecar(util.StashContainer))

			By("Waiting for Repository CRD")
			f.EventuallyRepository(&ss).Should(WithTransform(func(repoList []*api.Repository) int {
				return len(repoList)
			}, BeNumerically("==", int(*ss.Spec.Replicas))))

			By("Waiting for backup to complete")
			f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
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
			XIt(`should backup existing StatefulSet`, shouldBackupExistingStatefulSet)
		})

		Context(`"S3" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForS3Backend()
				restic = f.ResticForS3Backend()
			})
			It(`should backup new StatefulSet`, shouldBackupNewStatefulSet)
			XIt(`should backup existing StatefulSet`, shouldBackupExistingStatefulSet)
		})

		Context(`"DO" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForDOBackend()
				restic = f.ResticForDOBackend()
			})
			It(`should backup new StatefulSet`, shouldBackupNewStatefulSet)
			XIt(`should backup existing StatefulSet`, shouldBackupExistingStatefulSet)
		})

		Context(`"GCS" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForGCSBackend()
				restic = f.ResticForGCSBackend()
			})
			It(`should backup new StatefulSet`, shouldBackupNewStatefulSet)
			XIt(`should backup existing StatefulSet`, shouldBackupExistingStatefulSet)
		})

		Context(`"Azure" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForAzureBackend()
				restic = f.ResticForAzureBackend()
			})
			It(`should backup new StatefulSet`, shouldBackupNewStatefulSet)
			XIt(`should backup existing StatefulSet`, shouldBackupExistingStatefulSet)
		})

		Context(`"Swift" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForSwiftBackend()
				restic = f.ResticForSwiftBackend()
			})
			It(`should backup new StatefulSet`, shouldBackupNewStatefulSet)
			XIt(`should backup existing StatefulSet`, shouldBackupExistingStatefulSet)
		})

		Context(`"B2" backend`, func() {
			BeforeEach(func() {
				cred = f.SecretForB2Backend()
				restic = f.ResticForB2Backend()
			})
			It(`should backup new StatefulSet`, shouldBackupNewStatefulSet)
			XIt(`should backup existing StatefulSet`, shouldBackupExistingStatefulSet)
		})
	})

	XDescribe("Changing StatefulSet labels", func() {
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

	XDescribe("Changing Restic selector", func() {
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

	XDescribe("Deleting restic for", func() {
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

	Describe("Stash Webhook for", func() {
		AfterEach(func() {
			f.DeleteStatefulSet(ss.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteRestic(secondRestic.ObjectMeta)
			f.DeleteService(svc.ObjectMeta)
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
			It("should mutate and backup new StatefulSet", shouldMutateAndBackupNewStatefulSet)
			It("should not mutate new StatefulSet if no restic select it", shouldNotMutateNewStatefulSet)
			It("should reject to create new StatefulSet if multiple restic select it", shouldRejectToCreateNewStatefulSet)
			XIt("should remove sidecar instantly if label change to match no restic", shouldRemoveSidecarInstantly)
			XIt("should add sidecar instantly if label change to match single restic", shouldAddSidecarInstantly)
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

		Context(`Single Replica`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				restic.Spec.Type = api.BackupOffline
				restic.Spec.Schedule = "@every 3m"
			})
			It(`should backup new StatefulSet`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				cronJobName := util.ScaledownCronPrefix + restic.Name
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

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&ss).Should(WithTransform(func(repoList []*api.Repository) int {
					return len(repoList)
				}, BeNumerically("==", int(*ss.Spec.Replicas))))

				By("Waiting for initial backup to complete")
				f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically("==", 1)))

				By("Ensuring initial backup is not taken by cronJob")
				backupCron, err := f.KubeClient.BatchV1beta1().CronJobs(restic.Namespace).Get(cronJobName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(backupCron.Status.LastScheduleTime).Should(BeNil())

				By("Waiting for 3 backup by cronJob to complete")
				start := time.Now()
				for i := 2; i <= 4; i++ {
					fmt.Printf("=============== Waiting for backup no: %d ============\n", i)
					f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically("==", i)))
				}
				elapsedTime := time.Since(start).Minutes()

				// backup is scheduled for every 3 minutes.
				// so 3 backup by cronJob should not take more than 9 minutes + some overhead.(let 1 minute overhead for each backup)
				Expect(elapsedTime).Should(BeNumerically("<=", 9+3))

				By("Waiting for backup event")
				repos, err := f.StashClient.StashV1alpha1().Repositories(restic.Namespace).List(metav1.ListOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(repos.Items).NotTo(BeEmpty())
				f.EventualEvent(repos.Items[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">", 1)))
			})
		})

		Context(`Multiple Replica`, func() {
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForHostPathLocalBackend()
				restic.Spec.Type = api.BackupOffline
				restic.Spec.Schedule = "@every 3m"
			})
			It(`should backup new StatefulSet`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				cronJobName := util.ScaledownCronPrefix + restic.Name
				By("Checking cron job created: " + cronJobName)
				Eventually(func() error {
					_, err := f.KubeClient.BatchV1beta1().CronJobs(restic.Namespace).Get(cronJobName, metav1.GetOptions{})
					return err
				}).Should(BeNil())

				By("Creating service " + svc.Name)
				err = f.CreateService(svc)
				Expect(err).NotTo(HaveOccurred())

				By("Creating StatefulSet " + ss.Name)
				ss.Spec.Replicas = types.Int32P(2)
				_, err = f.CreateStatefulSet(ss)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for init-container")
				f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveInitContainer(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&ss).Should(WithTransform(func(repoList []*api.Repository) int {
					return len(repoList)
				}, BeNumerically("==", int(*ss.Spec.Replicas))))

				By("Waiting for initial backup to complete")
				f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically("==", 1)))

				By("Ensuring initial backup is not taken by cronJob")
				backupCron, err := f.KubeClient.BatchV1beta1().CronJobs(restic.Namespace).Get(cronJobName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(backupCron.Status.LastScheduleTime).Should(BeNil())

				By("Waiting for 3 backup by cronJob to complete")
				start := time.Now()
				for i := 2; i <= 4; i++ {
					fmt.Printf("=============== Waiting for backup no: %d ============\n", i)
					f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically("==", i)))
				}
				elapsedTime := time.Since(start).Minutes()

				// backup is scheduled for every 3 minutes.
				// so 3 backup by cronJob should not take more than 9 minutes + some overhead.(let 1 minute overhead for each backup)
				Expect(elapsedTime).Should(BeNumerically("<=", 9+3))

				By("Waiting for backup event")
				repos := f.StatefulSetRepos(&ss)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">", 1)))
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

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&ss).Should(WithTransform(func(repoList []*api.Repository) int {
					return len(repoList)
				}, BeNumerically("==", int(*ss.Spec.Replicas))))

				By("Waiting for backup to complete")
				f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.StatefulSetRepos(&ss)
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
				repos = f.StatefulSetRepos(&ss)
				Expect(repos).NotTo(BeEmpty())

				previousBackupCount := repos[0].Status.BackupCount

				By("Waiting 2 minutes")
				time.Sleep(2 * time.Minute)

				By("Checking that Backup count has not changed")
				repos = f.StatefulSetRepos(&ss)
				Expect(repos).NotTo(BeEmpty())
				Expect(repos[0].Status.BackupCount).Should(BeNumerically("==", previousBackupCount))

				By(`Patching Restic with "paused: false"`)
				err = f.CreateOrPatchRestic(restic.ObjectMeta, func(in *api.Restic) *api.Restic {
					in.Spec.Paused = false
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">", previousBackupCount)))

				By("Waiting for backup event")
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">", previousBackupCount)))
			})

		})
	})

	Describe("Create Repository CRD", func() {
		Context(`"Local" backend, single replica`, func() {
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
			It(`create`, func() {
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

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&ss).Should(WithTransform(func(repoList []*api.Repository) int {
					return len(repoList)
				}, BeNumerically("==", int(*ss.Spec.Replicas))))

				By("Waiting for backup to complete")
				f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.StatefulSetRepos(&ss)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
			})

		})
		Context(`"Local" backend, multiple replica`, func() {
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
			It(`create`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating service " + svc.Name)
				err = f.CreateService(svc)
				Expect(err).NotTo(HaveOccurred())

				By("Creating StatefulSet with 3 replica" + ss.Name)
				ss.Spec.Replicas = types.Int32P(3)
				_, err = f.CreateStatefulSet(ss)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&ss).Should(WithTransform(func(repoList []*api.Repository) int {
					return len(repoList)
				}, BeNumerically("==", int(*ss.Spec.Replicas))))

				By("Waiting for backup to complete")
				f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.StatefulSetRepos(&ss)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
			})

		})
	})

	Describe("Complete Recovery", func() {
		Context(`"Local" backend, single fileGroup`, func() {
			AfterEach(func() {
				f.CleanupRecoveredVolume(ss.ObjectMeta)
				f.DeleteStatefulSet(ss.ObjectMeta)
				f.DeleteRestic(restic.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
				f.DeleteRecovery(recovery.ObjectMeta)

				err := framework.WaitUntilRecoveryDeleted(f.StashClient, recovery.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
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

				By("Creating StatefulSet " + ss.Name)
				_, err = f.CreateStatefulSet(ss)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&ss).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.StatefulSetRepos(&ss)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(ss.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic))
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting ss")
				f.DeleteStatefulSet(ss.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// wait until statefulset terminated
				err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, ss.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, ss.Namespace, f.AppLabel())
				Expect(err).NotTo(HaveOccurred())

				repos = f.StatefulSetRepos(&ss)
				Expect(repos).NotTo(BeEmpty())
				repoLabeData, err := util.ExtractDataFromRepositoryLabel(repos[0].Labels)
				Expect(err).ShouldNot(HaveOccurred())
				recovery.Spec.Repository.Name = localRef.GetRepositoryCRDName(repoLabeData.PodName, repoLabeData.NodeName)
				recovery.Spec.Repository.Namespace = f.Namespace()

				By("Creating recovery " + recovery.Name)
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying ss with recovered volume")
				ss.Spec.Template.Spec.Volumes = f.RecoveredVolume()

				_, err = f.CreateStatefulSet(ss)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(ss.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic)).Should(BeEquivalentTo(previousData))
			})

		})

		Context(`"Local" backend, multiple fileGroup`, func() {
			AfterEach(func() {
				f.CleanupRecoveredVolume(ss.ObjectMeta)
				f.DeleteStatefulSet(ss.ObjectMeta)
				f.DeleteRestic(restic.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
				f.DeleteRecovery(recovery.ObjectMeta)
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

				ss.Spec.Template.Spec.Volumes = f.HostPathVolumeWithMultipleDirectory()
				By("Creating StatefulSet " + ss.Name)
				_, err = f.CreateStatefulSet(ss)
				Expect(err).NotTo(HaveOccurred())
				apps_util.WaitUntilStatefulSetReady(f.KubeClient, ss.ObjectMeta)

				By("Creating demo data in hostPath")
				err = f.CreateDemoData(ss.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic")
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				// delete old statefulset create new one so that it start with init container
				f.DeleteStatefulSet(ss.ObjectMeta)
				framework.WaitUntilStatefulSetDeleted(f.KubeClient, ss.ObjectMeta)
				_, err = f.CreateStatefulSet(ss)

				Expect(err).NotTo(HaveOccurred())
				apps_util.WaitUntilStatefulSetReady(f.KubeClient, ss.ObjectMeta)

				By("Waiting for init-container")
				f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&ss).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.StatefulSetRepos(&ss)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(ss.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic))
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting ss")
				f.DeleteStatefulSet(ss.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// wait until statefulset terminated
				err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, ss.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, ss.Namespace, f.AppLabel())
				Expect(err).NotTo(HaveOccurred())

				repos = f.StatefulSetRepos(&ss)
				Expect(repos).NotTo(BeEmpty())
				repoLabeData, err := util.ExtractDataFromRepositoryLabel(repos[0].Labels)
				Expect(err).ShouldNot(HaveOccurred())
				recovery.Spec.Repository.Name = localRef.GetRepositoryCRDName(repoLabeData.PodName, repoLabeData.NodeName)
				recovery.Spec.Repository.Namespace = f.Namespace()

				By("Creating recovery " + recovery.Name)
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying ss with recovered volume")
				ss.Spec.Template.Spec.Volumes = f.RecoveredVolume()

				_, err = f.CreateStatefulSet(ss)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(ss.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic)).Should(BeEquivalentTo(previousData))
			})

		})
	})

	Describe("Recover from different namespace", func() {
		var (
			recoveryNamespace *core.Namespace
		)
		Context(`"Local" backend, single fileGroup`, func() {
			AfterEach(func() {
				f.CleanupRecoveredVolume(ss.ObjectMeta)
				f.DeleteStatefulSet(ss.ObjectMeta)
				f.DeleteRestic(restic.ObjectMeta)
				f.DeleteSecret(cred.ObjectMeta)
				f.DeleteRecovery(recovery.ObjectMeta)
				f.DeleteNamespace(recoveryNamespace.Name)

				err := framework.WaitUntilRecoveryDeleted(f.StashClient, recovery.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = framework.WaitUntilNamespaceDeleted(f.KubeClient, recoveryNamespace.ObjectMeta)
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

				By("Creating StatefulSet " + ss.Name)
				_, err = f.CreateStatefulSet(ss)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for Repository CRD")
				f.EventuallyRepository(&ss).ShouldNot(BeEmpty())

				By("Waiting for backup to complete")
				f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				repos := f.StatefulSetRepos(&ss)
				Expect(repos).NotTo(BeEmpty())
				f.EventualEvent(repos[0].ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))

				By("Reading data from /source/data mountPath")
				previousData, err := f.ReadDataFromMountedDir(ss.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic))
				Expect(err).NotTo(HaveOccurred())
				Expect(previousData).NotTo(BeEmpty())

				By("Deleting ss")
				f.DeleteStatefulSet(ss.ObjectMeta)

				By("Deleting restic")
				f.DeleteRestic(restic.ObjectMeta)

				// wait until statefulset terminated
				err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, ss.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = framework.WaitUntilResticDeleted(f.StashClient, restic.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = core_util.WaitUntillPodTerminatedByLabel(f.KubeClient, ss.Namespace, f.AppLabel())
				Expect(err).NotTo(HaveOccurred())

				repos = f.StatefulSetRepos(&ss)
				Expect(repos).NotTo(BeEmpty())
				repoLabeData, err := util.ExtractDataFromRepositoryLabel(repos[0].Labels)
				Expect(err).ShouldNot(HaveOccurred())
				recovery.Spec.Repository.Name = localRef.GetRepositoryCRDName(repoLabeData.PodName, repoLabeData.NodeName)
				recovery.Spec.Repository.Namespace = f.Namespace()

				By("Creating new namespace: " + recoveryNamespace.Name)
				err = f.CreateNamespace(recoveryNamespace)
				Expect(err).NotTo(HaveOccurred())

				By("Creating recovery " + recovery.Name)
				recovery.Name = recoveryNamespace.Name
				err = f.CreateRecovery(recovery)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for recovery succeed")
				f.EventuallyRecoverySucceed(recovery.ObjectMeta).Should(BeTrue())

				By("Checking cleanup")
				f.DeleteJobAndDependents(util.RecoveryJobPrefix+recovery.Name, &recovery)

				By("Re-deploying ss with recovered volume")
				ss.Namespace = recoveryNamespace.Name
				ss.Spec.Template.Spec.Volumes = f.RecoveredVolume()

				_, err = f.CreateStatefulSet(ss)
				Expect(err).NotTo(HaveOccurred())

				By("Reading data from /source/data mountPath")
				f.EventuallyRecoveredData(ss.ObjectMeta, framework.GetPathsFromResticFileGroups(&restic)).Should(BeEquivalentTo(previousData))
			})

		})
	})
})
