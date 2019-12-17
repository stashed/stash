/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package hooks

import (
	"database/sql"
	"fmt"
	"time"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	_ "github.com/go-sql-driver/mysql"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	pfutil "kmodules.xyz/client-go/tools/portforward"
	probev1 "kmodules.xyz/prober/api/v1"
)

var _ = Describe("PreBackup Hook", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			f.PrintDebugHelpers()
		}
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
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PreBackup: &probev1.Handler{
									HTTPGet: &core.HTTPGetAction{
										Scheme: "HTTP",
										Host:   fmt.Sprintf("%s-0.%s.%s.svc", statefulset.Name, framework.TEST_HEADLESS_SERVICE, f.Namespace()),
										Path:   "/success",
										Port:   intstr.FromInt(framework.HttpPort),
									},
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
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

						// Setup Backup
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PreBackup: &probev1.Handler{
									HTTPGet: &core.HTTPGetAction{
										Scheme: "HTTP",
										Path:   "/success",
										Port:   intstr.FromString(framework.HttpPortName),
									},
									ContainerName: framework.ProberDemoPodPrefix,
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
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

					// Setup Backup
					backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
						bc.Spec.Hooks = &v1beta1.BackupHooks{
							PreBackup: &probev1.Handler{
								HTTPGet: &core.HTTPGetAction{
									Scheme: "HTTP",
									Path:   "/fail",
									Port:   intstr.FromString(framework.HttpPortName),
								},
								ContainerName: framework.ProberDemoPodPrefix,
							},
						}
					})
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup of the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has failed")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))

					By("Verifying that Repository has zero SnapshotCount")
					repo2, err := f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Get(repo.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(repo2.Status.SnapshotCount).Should(BeZero())

					By("Verifying that no bucket has been created in the backend")
					_, err = f.BrowseMinioRepository(repo)
					// if the bucket does not exist, then it should return an error with "not found" message
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(BeEquivalentTo("not found"))
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
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PreBackup: &probev1.Handler{
									HTTPPost: &probev1.HTTPPostAction{
										Scheme: "HTTP",
										Host:   fmt.Sprintf("%s-0.%s.%s.svc", statefulset.Name, framework.TEST_HEADLESS_SERVICE, f.Namespace()),
										Path:   "/post-demo",
										Port:   intstr.FromInt(framework.HttpPort),
									},
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
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

						// Setup Backup
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PreBackup: &probev1.Handler{
									HTTPPost: &probev1.HTTPPostAction{
										Scheme: "HTTP",
										Path:   "/post-demo",
										Port:   intstr.FromString(framework.HttpPortName),
									},
									ContainerName: framework.ProberDemoPodPrefix,
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
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

						// Setup Backup
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PreBackup: &probev1.Handler{
									HTTPPost: &probev1.HTTPPostAction{
										Scheme: "HTTP",
										Path:   "/post-demo",
										Port:   intstr.FromString(framework.HttpPortName),
										Body:   `{"expectedCode":"200","expectedResponse":"success"}`,
									},
									ContainerName: framework.ProberDemoPodPrefix,
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
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

						// Setup Backup
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PreBackup: &probev1.Handler{
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
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
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

					// Setup Backup
					backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
						bc.Spec.Hooks = &v1beta1.BackupHooks{
							PreBackup: &probev1.Handler{
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
							},
						}
					})
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup of the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has failed")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))

					By("Verifying that Repository has zero SnapshotCount")
					repo2, err := f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Get(repo.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(repo2.Status.SnapshotCount).Should(BeZero())

					By("Verifying that no bucket has been created in the backend")
					_, err = f.BrowseMinioRepository(repo)
					// if the bucket does not exist, then it should return an error with "not found" message
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(BeEquivalentTo("not found"))
				})
			})
		})
	})

	Context("TCPSocketAction", func() {
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
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PreBackup: &probev1.Handler{
									TCPSocket: &core.TCPSocketAction{
										Host: fmt.Sprintf("%s-0.%s.%s.svc", statefulset.Name, framework.TEST_HEADLESS_SERVICE, f.Namespace()),
										Port: intstr.FromInt(framework.TcpPort),
									},
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
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

						// Setup Backup
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PreBackup: &probev1.Handler{
									TCPSocket: &core.TCPSocketAction{
										Port: intstr.FromString(framework.TcpPortName),
									},
									ContainerName: framework.ProberDemoPodPrefix,
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
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

					// Setup Backup
					backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
						bc.Spec.Hooks = &v1beta1.BackupHooks{
							PreBackup: &probev1.Handler{
								TCPSocket: &core.TCPSocketAction{
									Port: intstr.FromInt(9091),
								},
								ContainerName: framework.ProberDemoPodPrefix,
							},
						}
					})
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup of the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has failed")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))

					By("Verifying that Repository has zero SnapshotCount")
					repo2, err := f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Get(repo.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(repo2.Status.SnapshotCount).Should(BeZero())

					By("Verifying that no bucket has been created in the backend")
					_, err = f.BrowseMinioRepository(repo)
					// if the bucket does not exist, then it should return an error with "not found" message
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(BeEquivalentTo("not found"))
				})
			})
		})
	})

	Context("ExecAction", func() {
		Context("Sidecar Model", func() {
			Context("Success Test", func() {
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

					// Setup Backup
					backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
						bc.Spec.Hooks = &v1beta1.BackupHooks{
							PreBackup: &probev1.Handler{
								Exec: &core.ExecAction{
									Command: []string{"/bin/sh", "-c", `exit $EXIT_CODE_SUCCESS`},
								},
								ContainerName: framework.ProberDemoPodPrefix,
							},
						}
					})
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup of the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has succeeded")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
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

					// Setup Backup
					backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
						bc.Spec.Hooks = &v1beta1.BackupHooks{
							PreBackup: &probev1.Handler{
								Exec: &core.ExecAction{
									Command: []string{"/bin/sh", "-c", `exit $EXIT_CODE_FAIL`},
								},
								ContainerName: framework.ProberDemoPodPrefix,
							},
						}
					})
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup of the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has failed")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))

					By("Verifying that Repository has zero SnapshotCount")
					repo2, err := f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Get(repo.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(repo2.Status.SnapshotCount).Should(BeZero())

					By("Verifying that no bucket has been created in the backend")
					_, err = f.BrowseMinioRepository(repo)
					// if the bucket does not exist, then it should return an error with "not found" message
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(BeEquivalentTo("not found"))
				})
			})
		})

		Context("Job Model", func() {
			Context("PVC", func() {
				Context("Success Cases", func() {
					It("should backup the file created in preBackup Hook", func() {
						// Create new PVC
						pvc, err := f.CreateNewPVC(fmt.Sprintf("source-pvc-%s", f.App()))
						Expect(err).NotTo(HaveOccurred())

						// Deploy a Pod
						pod, err := f.DeployPod(pvc.Name)
						Expect(err).NotTo(HaveOccurred())

						// Generate Sample Data
						sampleData, err := f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						// Setup PVC Backup
						backupConfig, err := f.SetupPVCBackup(pvc, repo, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PreBackup: &probev1.Handler{
									Exec: &core.ExecAction{
										Command: []string{"touch", fmt.Sprintf("%s/pre-hook.txt", apis.StashDefaultMountPath)},
									},
									ContainerName: apis.PreTaskHook,
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has succeeded")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

						By("Reading data after executing preBackup hook")
						newData, err := f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())
						Expect(newData).ShouldNot(BeSameAs(sampleData))

						// Simulate disaster scenario. Delete the data from source PVC
						By("Deleting sample data from source Pod")
						err = f.CleanupSampleDataFromWorkload(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())

						// Restore the backed up data
						By("Restoring the backed up data")
						restoreSession, err := f.SetupRestoreProcessForPVC(pvc, repo)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that RestoreSession succeeded")
						completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

						// Get restored data
						restoredData := f.RestoredData(pod.ObjectMeta, apis.KindPod)

						// Verify that restored data is same as the original data
						By("Verifying restored data is same as the data after executing preBackup hook")
						Expect(restoredData).Should(BeSameAs(newData))
					})
				})

				Context("Failure Cases", func() {
					It("should not take backup when the preBackup hook failed", func() {
						// Create new PVC
						pvc, err := f.CreateNewPVC(fmt.Sprintf("source-pvc-%s", f.App()))
						Expect(err).NotTo(HaveOccurred())

						// Deploy a Pod
						pod, err := f.DeployPod(pvc.Name)
						Expect(err).NotTo(HaveOccurred())

						// Generate Sample Data
						_, err = f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						// Setup PVC Backup
						backupConfig, err := f.SetupPVCBackup(pvc, repo, func(bc *v1beta1.BackupConfiguration) {
							// try to write a file in an invalid directory so that the hook fail.
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PreBackup: &probev1.Handler{
									Exec: &core.ExecAction{
										Command: []string{"touch", "/invalid/directory/pre-hook.txt"},
									},
									ContainerName: apis.PreTaskHook,
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has failed")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))

						By("Verifying that Repository has zero SnapshotCount")
						repo2, err := f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Get(repo.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(repo2.Status.SnapshotCount).Should(BeZero())

						By("Verifying that no bucket has been created in the backend")
						_, err = f.BrowseMinioRepository(repo)
						// if the bucket does not exist, then it should return an error with "not found" message
						Expect(err).Should(HaveOccurred())
						Expect(err.Error()).Should(BeEquivalentTo("not found"))
					})
				})
			})

			Context("MySQL", func() {
				const (
					sampleTable = "StashDemo"
				)

				BeforeEach(func() {
					// Skip test if respective Functions and Tasks are not installed.
					if !f.MySQLAddonInstalled() {
						Skip("MySQL addon is not installed")
					}
				})

				Context("Success Test", func() {
					It("should make the database read-only in preBackup hook", func() {
						// Deploy MySQL database and respective service,secret,PVC and AppBinding.
						By("Deploying MySQL Server")
						dpl, appBinding, err := f.DeployMySQLDatabase()
						Expect(err).NotTo(HaveOccurred())

						By("Port forwarding MySQL pod")
						pod, err := f.GetPod(dpl.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())
						tunnel := pfutil.NewTunnel(f.KubeClient.CoreV1().RESTClient(), f.ClientConfig, pod.Namespace, pod.Name, framework.MySQLServingPortNumber)
						defer tunnel.Close()
						err = tunnel.ForwardPort()
						Expect(err).NotTo(HaveOccurred())

						By("Connecting with MySQL Server")
						connstr := fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql", framework.SuperUser, f.App(), framework.LocalHostIP, tunnel.Local)
						db, err := sql.Open("mysql", connstr)
						Expect(err).NotTo(HaveOccurred())
						defer db.Close()
						db.SetConnMaxLifetime(time.Second * 10)
						err = f.EventuallyConnectWithMySQLServer(db)
						Expect(err).NotTo(HaveOccurred())

						By("Creating Sample Table")
						err = f.CreateTable(db, sampleTable)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that the sample table has been created")
						tables, err := f.ListTables(db)
						Expect(err).NotTo(HaveOccurred())
						Expect(tables.Has(sampleTable)).Should(BeTrue())

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						// Setup Database Backup
						// Here, we are going to make the database read-only in preBackup hook.
						// We won't make the database writable after the backup because we will try to write
						// in the read only database to verify that the preBackup hook was executed properly
						backupConfig, err := f.SetupDatabaseBackup(appBinding, repo, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PreBackup: &probev1.Handler{
									Exec: &core.ExecAction{
										Command: []string{"/bin/sh", "-c",
											`mysql -u root --password=$MYSQL_ROOT_PASSWORD -e "SET GLOBAL super_read_only = ON;"`},
									},
									ContainerName: framework.MySQLContainerName,
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has succeeded")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

						By("Verifying that the database is read-only")
						err = f.CreateTable(db, "readOnlyTest")
						Expect(err).Should(HaveOccurred())
					})
				})

				Context("Failure Test", func() {
					It("should not take backup when preBackup hook failed", func() {
						// Deploy MySQL database and respective service,secret,PVC and AppBinding.
						By("Deploying MySQL Server")
						dpl, appBinding, err := f.DeployMySQLDatabase()
						Expect(err).NotTo(HaveOccurred())

						By("Port forwarding MySQL pod")
						pod, err := f.GetPod(dpl.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())
						tunnel := pfutil.NewTunnel(f.KubeClient.CoreV1().RESTClient(), f.ClientConfig, pod.Namespace, pod.Name, framework.MySQLServingPortNumber)
						defer tunnel.Close()
						err = tunnel.ForwardPort()
						Expect(err).NotTo(HaveOccurred())

						By("Connecting with MySQL Server")
						connstr := fmt.Sprintf("%s:%s@tcp(%s:%d)/mysql", framework.SuperUser, f.App(), framework.LocalHostIP, tunnel.Local)
						db, err := sql.Open("mysql", connstr)
						Expect(err).NotTo(HaveOccurred())
						defer db.Close()
						db.SetConnMaxLifetime(time.Second * 10)
						err = f.EventuallyConnectWithMySQLServer(db)
						Expect(err).NotTo(HaveOccurred())

						By("Creating Sample Table")
						err = f.CreateTable(db, sampleTable)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that the sample table has been created")
						tables, err := f.ListTables(db)
						Expect(err).NotTo(HaveOccurred())
						Expect(tables.Has(sampleTable)).Should(BeTrue())

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						// Setup Database Backup
						// Return non-zero exit code from the preBackup hook so that it fail.
						backupConfig, err := f.SetupDatabaseBackup(appBinding, repo, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PreBackup: &probev1.Handler{
									Exec: &core.ExecAction{
										Command: []string{"/bin/sh", "-c", "exit 1"},
									},
									ContainerName: framework.MySQLContainerName,
								},
							}
						})
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has failed")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))

						By("Verifying that Repository has zero SnapshotCount")
						repo2, err := f.StashClient.StashV1alpha1().Repositories(repo.Namespace).Get(repo.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(repo2.Status.SnapshotCount).Should(BeZero())

						By("Verifying that no bucket has been created in the backend")
						_, err = f.BrowseMinioRepository(repo)
						// if the bucket does not exist, then it should return an error with "not found" message
						Expect(err).Should(HaveOccurred())
						Expect(err.Error()).Should(BeEquivalentTo("not found"))
					})
				})
			})
		})
	})
})
