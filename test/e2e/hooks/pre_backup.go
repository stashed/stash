package hooks

import (
	"fmt"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	probev1 "kmodules.xyz/prober/api/v1"
)

var _ = Describe("PreBackup Hook", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("HTTPGetAction", func() {
		Context("Sidecar Model", func() {
			Context("Success Test", func() {
				Context("Host and Port specified", func() {
					It("should execute probe successfully", func() {
						// Deploy a StatefulSet with prober client. Here, we are using a StatefulSet because we need a stable address
						// for pod where http request will be sent.
						statefulset, err := f.DeployStatefulSetWithProbeClient(fmt.Sprintf("%s-%s", framework.ProberDemoPodPrefix, f.App()))
						Expect(err).NotTo(HaveOccurred())

						// Generate Sample Data
						_, err = f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
						Expect(err).NotTo(HaveOccurred())

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						// Setup workload Backup
						preBackupHook := &probev1.Handler{
							HTTPGet: &core.HTTPGetAction{
								Scheme: "HTTP",
								Host:   fmt.Sprintf("%s-0.%s.%s.svc", statefulset.Name, framework.TEST_HEADLESS_SERVICE, f.Namespace()),
								Path:   "/success",
								Port:   intstr.FromInt(framework.HttpPort),
							},
						}
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, preBackupHook, nil)
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has succeeded")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
					})
				})

				Context("Host and Port from Pod", func() {
					It("should execute probe successfully", func() {
						// Deploy a StatefulSet.
						statefulset, err := f.DeployStatefulSetWithProbeClient(fmt.Sprintf("%s-%s", framework.ProberDemoPodPrefix, f.App()))
						Expect(err).NotTo(HaveOccurred())

						// Generate Sample Data
						_, err = f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
						Expect(err).NotTo(HaveOccurred())

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						preBackupHook := &probev1.Handler{
							HTTPGet: &core.HTTPGetAction{
								Scheme: "HTTP",
								Path:   "/success",
								Port:   intstr.FromString(framework.HttpPortName),
							},
							ContainerName: framework.ProberDemoPodPrefix,
						}
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, preBackupHook, nil)
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has succeeded")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
					})
				})
			})

			Context("Failure Test", func() {
				It("should not take backup when probe failed", func() {
					// Deploy a StatefulSet.
					statefulset, err := f.DeployStatefulSetWithProbeClient(fmt.Sprintf("%s-%s", framework.ProberDemoPodPrefix, f.App()))
					Expect(err).NotTo(HaveOccurred())

					// Generate Sample Data
					_, err = f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())

					// Setup a Minio Repository
					repo, err := f.SetupMinioRepository()
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(repo)

					preBackupHook := &probev1.Handler{
						HTTPGet: &core.HTTPGetAction{
							Scheme: "HTTP",
							Path:   "/fail",
							Port:   intstr.FromString(framework.HttpPortName),
						},
						ContainerName: framework.ProberDemoPodPrefix,
					}
					backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, preBackupHook, nil)
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has failed")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))

					By("Verifying that no snapshot has been taken")
					repo, err = f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Get(repo.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(repo.Status.SnapshotCount).Should(BeZero())
				})
			})
		})
	})

	Context("HTTPPostAction", func() {
		Context("Sidecar Model", func() {
			Context("Success Test", func() {
				Context("Host and Port specified", func() {
					It("should execute probe successfully", func() {
						// Deploy a StatefulSet with prober client. Here, we are using a StatefulSet because we need a stable address
						// for pod where http request will be sent.
						statefulset, err := f.DeployStatefulSetWithProbeClient(fmt.Sprintf("%s-%s", framework.ProberDemoPodPrefix, f.App()))
						Expect(err).NotTo(HaveOccurred())

						// Generate Sample Data
						_, err = f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
						Expect(err).NotTo(HaveOccurred())

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						// Setup workload Backup
						preBackupHook := &probev1.Handler{
							HTTPPost: &probev1.HTTPPostAction{
								Scheme: "HTTP",
								Host:   fmt.Sprintf("%s-0.%s.%s.svc", statefulset.Name, framework.TEST_HEADLESS_SERVICE, f.Namespace()),
								Path:   "/post-demo",
								Port:   intstr.FromInt(framework.HttpPort),
							},
						}
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, preBackupHook, nil)
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has succeeded")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
					})
				})

				Context("Host and Port from Pod", func() {
					It("should execute probe successfully", func() {
						// Deploy a StatefulSet.
						statefulset, err := f.DeployStatefulSetWithProbeClient(fmt.Sprintf("%s-%s", framework.ProberDemoPodPrefix, f.App()))
						Expect(err).NotTo(HaveOccurred())

						// Generate Sample Data
						_, err = f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
						Expect(err).NotTo(HaveOccurred())

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						preBackupHook := &probev1.Handler{
							HTTPPost: &probev1.HTTPPostAction{
								Scheme: "HTTP",
								Path:   "/post-demo",
								Port:   intstr.FromString(framework.HttpPortName),
							},
							ContainerName: framework.ProberDemoPodPrefix,
						}
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, preBackupHook, nil)
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has succeeded")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
					})
				})

				Context("Json Data in Request Body", func() {
					It("server should echo the 'expectedCode' and 'expectedResponse' passed in the json body", func() {
						// Deploy a StatefulSet.
						statefulset, err := f.DeployStatefulSetWithProbeClient(fmt.Sprintf("%s-%s", framework.ProberDemoPodPrefix, f.App()))
						Expect(err).NotTo(HaveOccurred())

						// Generate Sample Data
						_, err = f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
						Expect(err).NotTo(HaveOccurred())

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						preBackupHook := &probev1.Handler{
							HTTPPost: &probev1.HTTPPostAction{
								Scheme: "HTTP",
								Path:   "/post-demo",
								Port:   intstr.FromString(framework.HttpPortName),
								Body:   `{"expectedCode":"200","expectedResponse":"success"}`,
							},
							ContainerName: framework.ProberDemoPodPrefix,
						}
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, preBackupHook, nil)
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has succeeded")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
					})
				})

				Context("Form Data in Request Body", func() {
					It("server should echo the 'expectedCode' and 'expectedResponse' passed as form data", func() {
						// Deploy a StatefulSet.
						statefulset, err := f.DeployStatefulSetWithProbeClient(fmt.Sprintf("%s-%s", framework.ProberDemoPodPrefix, f.App()))
						Expect(err).NotTo(HaveOccurred())

						// Generate Sample Data
						_, err = f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
						Expect(err).NotTo(HaveOccurred())

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						preBackupHook := &probev1.Handler{
							HTTPPost: &probev1.HTTPPostAction{
								Scheme: "HTTP",
								Path:   "/post-demo",
								Port:   intstr.FromString(framework.HttpPortName),
								Form: []probev1.FormEntry{
									{
										Key:    "expectedResponse",
										Values: []string{"success"},
									},
									{
										Key:    "expectedCode",
										Values: []string{"202"},
									},
								},
							},
							ContainerName: framework.ProberDemoPodPrefix,
						}
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, preBackupHook, nil)
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has succeeded")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
					})
				})
			})

			Context("Failure Test", func() {
				It("should not take backup when probe failed", func() {
					// Deploy a StatefulSet.
					statefulset, err := f.DeployStatefulSetWithProbeClient(fmt.Sprintf("%s-%s", framework.ProberDemoPodPrefix, f.App()))
					Expect(err).NotTo(HaveOccurred())

					// Generate Sample Data
					_, err = f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())

					// Setup a Minio Repository
					repo, err := f.SetupMinioRepository()
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(repo)

					preBackupHook := &probev1.Handler{
						HTTPPost: &probev1.HTTPPostAction{
							Scheme: "HTTP",
							Path:   "/post-demo",
							Port:   intstr.FromString(framework.HttpPortName),
							Form: []probev1.FormEntry{
								{
									Key:    "expectedResponse",
									Values: []string{"fail"},
								},
								{
									Key:    "expectedCode",
									Values: []string{"403"},
								},
							},
						},
						ContainerName: framework.ProberDemoPodPrefix,
					}
					backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, preBackupHook, nil)
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has failed")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))

					By("Verifying that no snapshot has been taken")
					repo, err = f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Get(repo.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(repo.Status.SnapshotCount).Should(BeZero())
				})
			})
		})
	})

})
