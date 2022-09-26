/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package misc

import (
	"time"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gomodules.xyz/pointer"
	"gomodules.xyz/x/arrays"
)

var _ = Describe("BackupSession", func() {
	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	JustAfterEach(func() {
		f.PrintDebugInfoOnFailure()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
		// StatefulSet's PVCs are not get cleanup by the CleanupTestResources() function.
		// Hence, we need to cleanup them manually.
		f.CleanupUndeletedPVCs()
	})

	Context("Backup History Limit", func() {
		It("should keep last X BackupSession", func() {
			historyLimit := int32(2)
			// Deploy a Deployment
			deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())

			// Generate Sample Data
			_, err = f.GenerateBigSampleFile(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Setup a Minio Repository
			repo, err := f.SetupMinioRepository()
			Expect(err).NotTo(HaveOccurred())

			// Setup workload Backup
			backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
				bc.Spec.BackupHistoryLimit = pointer.Int32P(historyLimit)
			})
			Expect(err).NotTo(HaveOccurred())

			var keepCandidates []string
			totalBS := 5
			By("Triggering multiple backups")
			for i := 0; i < totalBS; i++ {
				backupSession, err := f.TriggerInstantBackup(backupConfig.ObjectMeta, v1beta1.BackupInvokerRef{
					Name: backupConfig.Name,
					Kind: v1beta1.ResourceKindBackupConfiguration,
				})
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(backupSession)
				// store only last X BackupSession name
				if i == 0 || i >= totalBS-int(historyLimit) {
					keepCandidates = append(keepCandidates, backupSession.Name)
				}
				time.Sleep(1 * time.Second)
			}

			By("Waiting for all BackupSession to complete")
			f.EventuallyRunningBackupCompleted(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration).Should(BeTrue())

			By("Verifying that the remaining BackupSessions are the desired ones")
			remainingBS, err := f.GetBackupSessionsForInvoker(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration)
			Expect(err).NotTo(HaveOccurred())
			for _, bs := range remainingBS.Items {
				exist, _ := arrays.Contains(keepCandidates, bs.Name)
				Expect(exist).To(BeTrue())
			}
		})
	})

	Context("Schedule Overlap", func() {
		It("should skip new session if previous one is running", func() {
			// Deploy a StatefulSet
			ss, err := f.DeployStatefulSet(framework.SourceStatefulSet, int32(3), framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())

			// Generate Sample Data
			_, err = f.GenerateBigSampleFile(ss.ObjectMeta, apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())

			// Setup a Minio Repository
			repo, err := f.SetupMinioRepository()
			Expect(err).NotTo(HaveOccurred())

			// Setup workload Backup
			backupConfig, err := f.SetupWorkloadBackup(ss.ObjectMeta, repo, apis.KindStatefulSet, func(bc *v1beta1.BackupConfiguration) {
				bc.Spec.BackupHistoryLimit = pointer.Int32P(5)
			})
			Expect(err).NotTo(HaveOccurred())

			totalBS := 5
			By("Triggering multiple backups")
			for i := 0; i < totalBS; i++ {
				backupSession, err := f.TriggerInstantBackup(backupConfig.ObjectMeta, v1beta1.BackupInvokerRef{
					Name: backupConfig.Name,
					Kind: v1beta1.ResourceKindBackupConfiguration,
				})
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(backupSession)
				time.Sleep(1 * time.Second)
			}

			By("Waiting for all BackupSession to complete")
			f.EventuallyRunningBackupCompleted(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration).Should(BeTrue())

			By("Verifying that the only one BackupSession has succeeded")
			successfulSessionCount, err := f.GetSuccessfulBackupSessionCount(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration)
			Expect(err).NotTo(HaveOccurred())
			Expect(successfulSessionCount).Should(BeNumerically("==", 1))

			By("Verifying that others BackupSession has been skipped")
			skippedSessionCount, err := f.GetSkippedBackupSessionCount(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration)
			Expect(err).NotTo(HaveOccurred())
			Expect(skippedSessionCount).Should(BeNumerically("==", totalBS-1))
		})
	})
})
