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
	"strings"

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
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apps_util "kmodules.xyz/client-go/apps/v1"
)

var _ = Describe("ReplicaSet", func() {

	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	var (
		createPVC = func(name string) *core.PersistentVolumeClaim {
			// Generate PVC definition
			pvc := f.PersistentVolumeClaim()
			pvc.Name = fmt.Sprintf("%s-pvc-%s", strings.Split(name, "-")[0], f.App())

			By("Creating PVC: " + pvc.Name)
			createdPVC, err := f.CreatePersistentVolumeClaim(pvc)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdPVC)

			return createdPVC
		}

		deployRS = func(name string) *apps.ReplicaSet {
			// Create PVC for ReplicaSet
			pvc := createPVC(name)
			// Generate ReplicaSet definition
			rs := f.ReplicaSet(pvc.Name)
			rs.Name = name

			By("Deploying ReplicaSet: " + rs.Name)
			createdRS, err := f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdRS)

			By("Waiting for ReplicaSet to be ready")
			err = apps_util.WaitUntilReplicaSetReady(f.KubeClient, createdRS.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdRS.ObjectMeta).Should(BeTrue())

			return createdRS
		}

		deployRSWithMultipleReplica = func(name string) *apps.ReplicaSet {
			// Create PVC for ReplicaSet
			pvc := createPVC(fmt.Sprintf("source-pvc-%s", f.App()))
			// Generate ReplicaSet definition
			rs := f.ReplicaSet(pvc.Name)
			rs.Spec.Replicas = types.Int32P(2) // two replicas
			rs.Name = name

			By("Deploying ReplicaSet: " + rs.Name)
			createdRS, err := f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdRS)

			By("Waiting for ReplicaSet to be ready")
			err = apps_util.WaitUntilReplicaSetReady(f.KubeClient, createdRS.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			// check that we can execute command to the pod.
			// this is necessary because we will exec into the pods and create sample data
			f.EventuallyPodAccessible(createdRS.ObjectMeta).Should(BeTrue())

			return createdRS
		}

		generateSampleData = func(rs *apps.ReplicaSet) sets.String {
			By("Generating sample data inside ReplicaSet pods")
			err := f.CreateSampleDataInsideWorkload(rs.ObjectMeta, apis.KindReplicaSet)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that sample data has been generated")
			sampleData, err := f.ReadSampleDataFromFromWorkload(rs.ObjectMeta, apis.KindReplicaSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleData).ShouldNot(BeEmpty())

			return sampleData
		}

		getTargetRef = func(rs *apps.ReplicaSet) v1beta1.TargetRef {
			return v1beta1.TargetRef{
				Name:       rs.Name,
				Kind:       apis.KindReplicaSet,
				APIVersion: "apps/v1",
			}
		}

		setupRSBackup = func(rs *apps.ReplicaSet, repo *api.Repository) *v1beta1.BackupConfiguration {
			// Generate desired BackupConfiguration definition
			backupConfig := f.GetBackupConfigurationForWorkload(repo.Name, getTargetRef(rs))

			By("Creating BackupConfiguration: " + backupConfig.Name)
			createdBC, err := f.StashClient.StashV1beta1().BackupConfigurations(backupConfig.Namespace).Create(backupConfig)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(createdBC)

			By("Verifying that backup triggering CronJob has been created")
			f.EventuallyCronJobCreated(backupConfig.ObjectMeta).Should(BeTrue())

			By("Verifying that sidecar has been injected")
			f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for ReplicaSet to be ready with sidecar")
			err = f.WaitUntilRSReadyWithSidecar(rs.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			return createdBC
		}

		takeInstantBackup = func(rs *apps.ReplicaSet, repo *api.Repository) {
			// Setup Backup
			backupConfig := setupRSBackup(rs, repo)

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

		restoreData = func(rs *apps.ReplicaSet, repo *api.Repository) sets.String {
			By("Creating RestoreSession")
			restoreSession := f.GetRestoreSessionForWorkload(repo.Name, getTargetRef(rs))
			err := f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(restoreSession)

			By("Verifying that init-container has been injected")
			f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveInitContainer(util.StashInitContainer))

			By("Waiting for restore process to complete")
			f.EventuallyRestoreProcessCompleted(restoreSession.ObjectMeta).Should(BeTrue())

			By("Verifying that RestoreSession succeeded")
			completedRestore, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedRestore.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

			By("Waiting for ReplicaSet to be ready with init-container")
			err = f.WaitUntilRSReadyWithInitContainer(rs.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			f.EventuallyPodAccessible(rs.ObjectMeta).Should(BeTrue())

			By("Reading restored data")
			restoredData, err := f.ReadSampleDataFromFromWorkload(rs.ObjectMeta, apis.KindReplicaSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(restoredData).NotTo(BeEmpty())

			return restoredData
		}
	)

	Context("ReplicaSet", func() {

		Context("Restore in same ReplicaSet", func() {

			It("should Backup & Restore in the source ReplicaSet", func() {
				// Deploy a ReplicaSet
				rs := deployRS(fmt.Sprintf("source-rs-%s", f.App()))

				// Generate Sample Data
				sampleData := generateSampleData(rs)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Take an Instant Backup the Sample Data
				takeInstantBackup(rs, repo)

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source ReplicaSet")
				err = f.CleanupSampleDataFromWorkload(rs.ObjectMeta, apis.KindReplicaSet)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original ReplicaSet")
				restoredData := restoreData(rs, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Restore in different ReplicaSet", func() {

			It("should restore backed up data into different ReplicaSet", func() {
				// Deploy a ReplicaSet
				rs := deployRS(fmt.Sprintf("source-rs-%s", f.App()))

				// Generate Sample Data
				sampleData := generateSampleData(rs)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Take an Instant Backup the Sample Data
				takeInstantBackup(rs, repo)

				// Deploy restored ReplicaSet
				restoredRS := deployRS(fmt.Sprintf("restored-rs-%s", f.App()))

				// Restore the backup data
				By("Restoring the backed up data in the restored ReplicaSet")
				restoredData := restoreData(restoredRS, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

		Context("Leader election for backup and restore ReplicaSet", func() {

			It("Should leader elect and backup and restore ReplicaSet", func() {
				// Deploy a ReplicaSet
				rs := deployRSWithMultipleReplica(fmt.Sprintf("source-rs-%s", f.App()))

				//  Generate Sample Data
				sampleData := generateSampleData(rs)

				// Setup a Minio Repository
				By("Creating Repository")
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup Backup
				backupConfig := setupRSBackup(rs, repo)

				By("Waiting for leader election")
				f.CheckLeaderElection(rs.ObjectMeta, apis.KindReplicaSet, v1beta1.ResourceKindBackupConfiguration)

				// Create Instant BackupSession for Trigger Instant Backup
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

				// Simulate disaster scenario. Delete the data from source PVC
				By("Deleting sample data from source ReplicaSet")
				err = f.CleanupSampleDataFromWorkload(rs.ObjectMeta, apis.KindReplicaSet)
				Expect(err).NotTo(HaveOccurred())

				// Restore the backup data
				By("Restoring the backed up data in the original ReplicaSet")
				restoredData := restoreData(rs, repo)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})

	})

})
