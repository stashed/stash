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
package workloads

import (
	"fmt"

	"stash.appscode.dev/stash/apis"
	api "stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	"github.com/appscode/go/sets"
	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apps_util "kmodules.xyz/client-go/apps/v1"
)

var _ = Describe("StatefulSet", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	var (
		deploySS = func(name string) *apps.StatefulSet {
			// Generate StatefulSet definition
			ss := f.StatefulSetForV1beta1API()
			ss.Name = name

			By("Deploying StatefulSet: " + ss.Name)
			createdss, err := f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdss)

			By("Waiting for StatefulSet to be ready")
			err = apps_util.WaitUntilStatefulSetReady(f.KubeClient, createdss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdss.ObjectMeta).Should(BeTrue())

			return createdss
		}

		deployScaledUpSS = func(name string) *apps.StatefulSet {
			// Generate StatefulSet definition
			ss := f.StatefulSetForV1beta1API()
			ss.Name = name
			// scaled up StatefulSet
			ss.Spec.Replicas = types.Int32P(5)

			By("Deploying StatefulSet: " + ss.Name)
			createdss, err := f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdss)

			By("Waiting for StatefulSet to be ready")
			err = apps_util.WaitUntilStatefulSetReady(f.KubeClient, createdss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdss.ObjectMeta).Should(BeTrue())

			return createdss
		}

		generateSampleData = func(ss *apps.StatefulSet) sets.String {
			By("Generating sample data inside StatefulSet pods")
			err := f.CreateSampleDataInsideWorkload(ss.ObjectMeta, apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that sample data has been generated")
			sampleData, err := f.ReadSampleDataFromFromWorkload(ss.ObjectMeta, apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleData).ShouldNot(BeEmpty())

			return sampleData
		}

		getTargetRef = func(ss *apps.StatefulSet) v1beta1.TargetRef {
			return v1beta1.TargetRef{
				Name:       ss.Name,
				Kind:       apis.KindStatefulSet,
				APIVersion: "apps/v1",
			}
		}

		setupSSBackup = func(ss *apps.StatefulSet, repo *api.Repository) *v1beta1.BackupConfiguration {
			// Generate desired BackupConfiguration definition
			backupConfig := f.GetBackupConfigurationForWorkload(repo.Name, getTargetRef(ss))

			By("Creating BackupConfiguration: " + backupConfig.Name)
			createdBC, err := f.StashClient.StashV1beta1().BackupConfigurations(backupConfig.Namespace).Create(backupConfig)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdBC)

			By("Verifying that backup triggering CronJob has been created")
			f.EventuallyCronJobCreated(backupConfig.ObjectMeta).Should(BeTrue())

			By("Verifying that sidecar has been injected")
			f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for StatefulSet to be ready with sidecar")
			err = f.WaitUntilStatefulSetReadyWithSidecar(ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			return createdBC
		}

		takeInstantBackup = func(ss *apps.StatefulSet, repo *api.Repository) {
			// Setup Backup
			backupConfig := setupSSBackup(ss, repo)

			// Trigger Instant Backup
			By("Triggering Instant Backup")
			backupSession, err := f.TriggerInstantBackup(backupConfig)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(backupSession)

			By("Waiting for backup process to complete")
			f.EventuallyBackupProcessCompleted(backupSession.ObjectMeta).Should(BeTrue())

			By("Verifying that BackupSession has succeeded")
			completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
		}

		restoreData = func(ss *apps.StatefulSet, repo *api.Repository) sets.String {
			By("Creating RestoreSession")
			restoreSession := f.GetRestoreSessionForWorkload(repo.Name, getTargetRef(ss))
			err := f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(restoreSession)

			By("Verifying that init-container has been injected")
			f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveInitContainer(util.StashInitContainer))

			By("Waiting for restore process to complete")
			f.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta).Should(BeTrue())

			By("Verifying that RestoreSession succeeded")
			completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

			By("Waiting for StatefulSet to be ready with init-container")
			err = f.WaitUntilStatefulSetWithInitContainer(ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			f.EventuallyPodAccessible(ss.ObjectMeta).Should(BeTrue())

			//fmt.Println("sleeping.......")
			//time.Sleep(2 * time.Minute)

			By("Reading restored data")
			restoredData, err := f.ReadSampleDataFromFromWorkload(ss.ObjectMeta, apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(restoredData).NotTo(BeEmpty())

			return restoredData
		}

		restoreDataOnScaledUpSS = func(ss *apps.StatefulSet, repo *api.Repository) sets.String {
			By("Creating RestoreSession")
			restoreSession := f.GetRestoreSessionForWorkload(repo.Name, getTargetRef(ss))
			restoreSession.Spec.Rules = []v1beta1.Rule{
				{
					TargetHosts: []string{
						"host-3",
						"host-4",
					},
					SourceHost: "host-0",
					Paths: []string{
						framework.TestSourceDataMountPath,
					},
				},
				{
					TargetHosts: []string{},
					SourceHost:  "",
					Paths: []string{
						framework.TestSourceDataMountPath,
					},
				},
			}
			err := f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(restoreSession)

			By("Verifying that init-container has been injected")
			f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveInitContainer(util.StashInitContainer))

			By("Waiting for restore process to complete")
			f.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta).Should(BeTrue())

			By("Verifying that RestoreSession succeeded")
			completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

			By("Waiting for StatefulSet to be ready with init-container")
			err = f.WaitUntilStatefulSetWithInitContainer(ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			f.EventuallyPodAccessible(ss.ObjectMeta).Should(BeTrue())

			By("Reading restored data")
			restoredData, err := f.ReadSampleDataFromFromWorkload(ss.ObjectMeta, apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(restoredData).NotTo(BeEmpty())

			return restoredData
		}
	)

	Context("StatefulSet", func() {

		Context("Restore in same StatefulSet", func() {

			It("should Backup & Restore in the source StatefulSet", func() {
				// Deploy a StatefulSet
				ss := deploySS(fmt.Sprintf("source-ss-%s", f.App()))

				// Generate Sample Data
				sampleData := generateSampleData(ss)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Take an Instant Backup the Sample Data
				takeInstantBackup(ss, repo)

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source StatefulSet")
				err = f.CleanupSampleDataFromWorkload(ss.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original Statefulset")
				restoredData := restoreData(ss, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(Equal(sampleData))
			})
		})

		Context("Restore in different StatefulSet", func() {

			It("should restore backed up data into different StatefulSet", func() {
				// Deploy a StatefulSet
				ss := deploySS(fmt.Sprintf("source-ss-%s", f.App()))

				// Generate Sample Data
				sampleData := generateSampleData(ss)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Take an Instant Backup the Sample Data
				takeInstantBackup(ss, repo)

				// Deploy restored StatefulSet
				restoredSS := deploySS(fmt.Sprintf("restored-ss-%s", f.App()))

				// Restore the backup data
				By("Restoring the backed up data in the restored StatefulSet")
				restoredData := restoreData(restoredSS, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(Equal(sampleData))
			})
		})

		Context("Restore on scaled up StatefulSet", func() {

			It("should restore backed up data into scaled up StatefulSet", func() {
				// Deploy a StatefulSet
				ss := deploySS(fmt.Sprintf("source-ss-%s", f.App()))

				// Generate Sample Data
				sampleData := generateSampleData(ss)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Take an Instant Backup the Sample Data
				takeInstantBackup(ss, repo)

				// Deploy restored StatefulSet
				restoredSS := deployScaledUpSS(fmt.Sprintf("restored-ss-%s", f.App()))

				// Restore the backup data
				By("Restoring the backed up data in the original StatefulSet")
				restoredData := restoreDataOnScaledUpSS(restoredSS, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

	})

})
