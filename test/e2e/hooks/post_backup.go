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
	"fmt"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"

	//. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	probev1 "kmodules.xyz/prober/api/v1"
)

var _ = Describe("PostBackup Hook", func() {

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
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PostBackup: &probev1.Handler{
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

						// Setup backup
						backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PostBackup: &probev1.Handler{
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
				It("should take a backup even when postBackup hook failed", func() {
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
							PostBackup: &probev1.Handler{
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

					By("Verifying that a backup has been taken")
					items, err := f.BrowseMinioRepository(repo)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(items).ShouldNot(BeEmpty())
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
								PostBackup: &probev1.Handler{
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
								PostBackup: &probev1.Handler{
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
								PostBackup: &probev1.Handler{
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
								PostBackup: &probev1.Handler{
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
				It("should take  a backup even when postBackup hook failed", func() {
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
							PostBackup: &probev1.Handler{
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

					By("Verifying that a backup has been taken")
					items, err := f.BrowseMinioRepository(repo)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(items).ShouldNot(BeEmpty())
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
								PostBackup: &probev1.Handler{
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
								PostBackup: &probev1.Handler{
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
				It("should take a backup even when postBackup hook failed", func() {
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
							PostBackup: &probev1.Handler{
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

					By("Verifying that a backup has been taken")
					items, err := f.BrowseMinioRepository(repo)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(items).ShouldNot(BeEmpty())
				})
			})
		})
	})

	Context("ExecAction", func() {
		Context("Sidecar Model", func() {
			Context("Success Test", func() {
				It("should cleanup the sample data in postBackup hook", func() {
					// Deploy a StatefulSet.
					statefulset, err := f.DeployStatefulSetWithProbeClient(fmt.Sprintf("%s-%s", framework.ProberDemoPodPrefix, f.App()))
					Expect(err).NotTo(HaveOccurred())

					// Read data at empty state
					emptyData, err := f.ReadSampleDataFromFromWorkload(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())

					// Generate Sample Data
					sampleData, err := f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())
					Expect(sampleData).ShouldNot(BeEquivalentTo(emptyData))

					// Setup a Minio Repository
					repo, err := f.SetupMinioRepository()
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(repo)

					// Setup Backup
					backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
						bc.Spec.Hooks = &v1beta1.BackupHooks{
							PostBackup: &probev1.Handler{
								Exec: &core.ExecAction{
									Command: []string{"/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", framework.TestSourceDataMountPath)},
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

					By("Verifying that data has been removed in postBackup hook")
					newData, err := f.ReadSampleDataFromFromWorkload(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())
					Expect(newData).Should(BeEquivalentTo(emptyData))
				})

				It("should execute postBackup hook even when backup process failed", func() {
					// Deploy a StatefulSet.
					statefulset, err := f.DeployStatefulSetWithProbeClient(fmt.Sprintf("%s-%s", framework.ProberDemoPodPrefix, f.App()))
					Expect(err).NotTo(HaveOccurred())

					// Read data at empty state
					emptyData, err := f.ReadSampleDataFromFromWorkload(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())

					// Generate Sample Data
					sampleData, err := f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())
					Expect(sampleData).ShouldNot(BeEquivalentTo(emptyData))

					// Setup a Minio Repository
					repo, err := f.SetupMinioRepository()
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(repo)

					// Setup Backup
					// Use invalid retentionPolicy so that the backup process fail in cleanup step
					// Remove old data in postBackup hook
					backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
						bc.Spec.Hooks = &v1beta1.BackupHooks{
							PostBackup: &probev1.Handler{
								Exec: &core.ExecAction{
									Command: []string{"/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", framework.TestSourceDataMountPath)},
								},
								ContainerName: framework.ProberDemoPodPrefix,
							},
						}
						bc.Spec.RetentionPolicy.KeepLast = 0 // invalid retention value to force backup process fail on cleanup step
					})
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup of the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has failed")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))

					By("Verifying that data has been removed in postBackup hook")
					newData, err := f.ReadSampleDataFromFromWorkload(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())
					Expect(newData).Should(BeEquivalentTo(emptyData))
				})
			})

			Context("Failure Test", func() {
				It("should take a backup even when postBackup probe failed", func() {
					// Deploy a StatefulSet.
					statefulset, err := f.DeployStatefulSetWithProbeClient(fmt.Sprintf("%s-%s", framework.ProberDemoPodPrefix, f.App()))
					Expect(err).NotTo(HaveOccurred())

					// Read data at empty state
					emptyData, err := f.ReadSampleDataFromFromWorkload(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())

					// Generate Sample Data
					sampleData, err := f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())
					Expect(sampleData).ShouldNot(BeEquivalentTo(emptyData))

					// Setup a Minio Repository
					repo, err := f.SetupMinioRepository()
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(repo)

					// Setup Backup
					// Return non-zero exit code so that the postBackup hook fail
					backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
						bc.Spec.Hooks = &v1beta1.BackupHooks{
							PostBackup: &probev1.Handler{
								Exec: &core.ExecAction{
									Command: []string{"/bin/sh", "-c", "exit 1"},
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

					By("Verifying that a backup has been taken")
					items, err := f.BrowseMinioRepository(repo)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(items).ShouldNot(BeEmpty())
				})
			})
		})

		Context("Job Model", func() {
			Context("PVC", func() {
				Context("Success Cases", func() {
					It("should cleanup the sample data in postBackup hook", func() {
						// Create new PVC
						pvc, err := f.CreateNewPVC(fmt.Sprintf("source-pvc-%s", f.App()))
						Expect(err).NotTo(HaveOccurred())

						// Deploy a Pod
						pod, err := f.DeployPod(pvc.Name)
						Expect(err).NotTo(HaveOccurred())

						// Read data at empty state
						emptyData, err := f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())

						// Generate Sample Data
						sampleData, err := f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())
						Expect(sampleData).NotTo(BeEquivalentTo(emptyData))

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						// Setup PVC Backup
						// Remove old data in postBackup hook
						backupConfig, err := f.SetupPVCBackup(pvc, repo, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PostBackup: &probev1.Handler{
									Exec: &core.ExecAction{
										Command: []string{"/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", apis.StashDefaultMountPath)},
									},
									ContainerName: apis.PostTaskHook,
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

						By("Verifying that data has been removed in postBackup hook")
						newData, err := f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())
						Expect(newData).Should(BeEquivalentTo(emptyData))
					})

					It("should execute postBackup hook even when backup failed", func() {
						// Create new PVC
						pvc, err := f.CreateNewPVC(fmt.Sprintf("source-pvc-%s", f.App()))
						Expect(err).NotTo(HaveOccurred())

						// Deploy a Pod
						pod, err := f.DeployPod(pvc.Name)
						Expect(err).NotTo(HaveOccurred())

						// Read data at empty state
						emptyData, err := f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())

						// Generate Sample Data
						sampleData, err := f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())
						Expect(sampleData).NotTo(BeEquivalentTo(emptyData))

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						// Setup PVC Backup
						// Use invalid retentionPolicy so that the backup process fail in cleanup step
						// Remove old data in postBackup hook
						backupConfig, err := f.SetupPVCBackup(pvc, repo, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PostBackup: &probev1.Handler{
									Exec: &core.ExecAction{
										Command: []string{"/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", apis.StashDefaultMountPath)},
									},
									ContainerName: apis.PostTaskHook,
								},
							}
							bc.Spec.RetentionPolicy.KeepLast = 0 // invalid retention value to force backup process fail on cleanup step
						})
						Expect(err).NotTo(HaveOccurred())

						// Take an Instant Backup of the Sample Data
						backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
						Expect(err).NotTo(HaveOccurred())

						By("Verifying that BackupSession has failed")
						completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))

						By("Verifying that data has been removed in postBackup hook")
						newData, err := f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())
						Expect(newData).Should(BeEquivalentTo(emptyData))
					})
				})

				Context("Failure Cases", func() {
					It("should take backup even when postBackup hook failed", func() {
						// Create new PVC
						pvc, err := f.CreateNewPVC(fmt.Sprintf("source-pvc-%s", f.App()))
						Expect(err).NotTo(HaveOccurred())

						// Deploy a Pod
						pod, err := f.DeployPod(pvc.Name)
						Expect(err).NotTo(HaveOccurred())

						// Read data at empty state
						emptyData, err := f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())

						// Generate Sample Data
						sampleData, err := f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)
						Expect(err).NotTo(HaveOccurred())
						Expect(sampleData).NotTo(BeEquivalentTo(emptyData))

						// Setup a Minio Repository
						repo, err := f.SetupMinioRepository()
						Expect(err).NotTo(HaveOccurred())
						f.AppendToCleanupList(repo)

						// Setup PVC Backup
						// Return non-zero exit code from postBackup hook so that it fail
						backupConfig, err := f.SetupPVCBackup(pvc, repo, func(bc *v1beta1.BackupConfiguration) {
							bc.Spec.Hooks = &v1beta1.BackupHooks{
								PostBackup: &probev1.Handler{
									Exec: &core.ExecAction{
										Command: []string{"/bin/sh", "-c", "exit 1"},
									},
									ContainerName: apis.PostTaskHook,
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

						By("Verifying that a backup has been taken")
						items, err := f.BrowseMinioRepository(repo)
						Expect(err).ShouldNot(HaveOccurred())
						Expect(items).ShouldNot(BeEmpty())
					})
				})
			})
		})
	})
})
