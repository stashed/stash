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
	"path/filepath"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	app_util "kmodules.xyz/client-go/apps/v1"
	probev1 "kmodules.xyz/prober/api/v1"
)

var _ = Describe("PreRestore Hook", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("ExecAction", func() {
		Context("Sidecar Model", func() {
			Context("Success Test", func() {
				It("probe should remove the corrupted data before restore", func() {
					// Deploy a StatefulSet with prober client. Here, we are using a StatefulSet because we need a stable address
					// for pod where http request will be sent.
					statefulset, err := f.DeployStatefulSetWithProbeClient(fmt.Sprintf("%s-%s", framework.ProberDemoPodPrefix, f.App()))
					Expect(err).NotTo(HaveOccurred())

					// Generate Sample Data
					sampleData, err := f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())

					// Setup a Minio Repository
					repo, err := f.SetupMinioRepository()
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(repo)

					// Setup workload Backup
					backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, nil, nil)
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has succeeded")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

					// Simulate disaster scenario. Corrupt the old data from source PVC.
					// Here, we are just creating a new file so that the resulting data does not match
					// with the sample data.
					By("Modifying source data")
					pod, err := f.GetPod(statefulset.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					_, err = f.ExecOnPod(pod, "touch", filepath.Join(framework.TestSourceDataMountPath, "corrupted-data.txt"))
					Expect(err).NotTo(HaveOccurred())

					preRestoreHook := &probev1.Handler{
						Exec: &core.ExecAction{
							Command: []string{"/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", framework.TestSourceDataMountPath)},
						},
						ContainerName: util.StashInitContainer,
					}

					// Restore the backed up data
					By("Restoring the backed up data in the original StatefulSet")
					restoreSession, err := f.SetupRestoreProcess(statefulset.ObjectMeta, repo, apis.KindStatefulSet, preRestoreHook, nil)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that RestoreSession succeeded")
					completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

					By("Verifying that original data has been restored")
					restoredData := f.RestoredData(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(restoredData).Should(BeSameAs(sampleData))
				})
			})

			Context("Failure Test", func() {
				It("should not restore when probe failed", func() {
					// Deploy a StatefulSet with prober client. Here, we are using a StatefulSet because we need a stable address
					// for pod where http request will be sent.
					statefulset, err := f.DeployStatefulSetWithProbeClient(fmt.Sprintf("%s-%s", framework.ProberDemoPodPrefix, f.App()))
					Expect(err).NotTo(HaveOccurred())

					// Generate Sample Data
					sampleData, err := f.GenerateSampleData(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(err).NotTo(HaveOccurred())

					// Setup a Minio Repository
					repo, err := f.SetupMinioRepository()
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(repo)

					// Setup workload Backup
					backupConfig, err := f.SetupWorkloadBackup(statefulset.ObjectMeta, repo, apis.KindStatefulSet, nil, nil)
					Expect(err).NotTo(HaveOccurred())

					// Take an Instant Backup the Sample Data
					backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that BackupSession has succeeded")
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

					// Simulate disaster scenario. Remove old data
					By("Removing source data")
					pod, err := f.GetPod(statefulset.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					_, err = f.ExecOnPod(pod, "/bin/sh", "-c", fmt.Sprintf("rm -rf %s/*", framework.TestSourceDataMountPath))
					Expect(err).NotTo(HaveOccurred())

					preRestoreHook := &probev1.Handler{
						Exec: &core.ExecAction{
							Command: []string{"/bin/sh", "-c", "exit 1"},
						},
						ContainerName: util.StashInitContainer,
					}

					// Restore the backed up data
					By("Restoring the backed up data in the original StatefulSet")
					restoreSession, err := f.SetupRestoreProcess(statefulset.ObjectMeta, repo, apis.KindStatefulSet, preRestoreHook, nil)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that RestoreSession has failed")
					completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionFailed))

					// Delete RestoreSession so that the StatefulSet can start normally
					By("Deleting RestoreSession")
					err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					// delete failed pod so that StatefulSet can start
					err = f.DeletePod(pod.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())
					err = app_util.WaitUntilStatefulSetReady(f.KubeClient, statefulset.ObjectMeta)
					Expect(err).NotTo(HaveOccurred())

					By("Verifying that no data has been restored")
					restoredData := f.RestoredData(statefulset.ObjectMeta, apis.KindStatefulSet)
					Expect(restoredData).ShouldNot(BeSameAs(sampleData))
				})
			})
		})
	})
})
