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
	"context"
	"fmt"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/stash/test/e2e/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gomodules.xyz/pointer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Pause Backup", func() {

	var f *framework.Invocation
	const AtEveryMinutes = "* * * * *"

	BeforeEach(func() {
		f = framework.NewInvocation()
	})

	JustAfterEach(func() {
		f.PrintDebugInfoOnFailure()
	})

	AfterEach(func() {
		err := f.CleanupTestResources()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Sidecar Model", func() {
		It("should pause scheduled backup but take instant backup", func() {
			// Deploy a Deployment
			deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())

			// Generate Sample Data
			_, err = f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			// Setup a Minio Repository
			repo, err := f.SetupMinioRepository()
			Expect(err).NotTo(HaveOccurred())

			// Setup workload Backup
			backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
				bc.Spec.Schedule = AtEveryMinutes
				bc.Spec.BackupHistoryLimit = pointer.Int32P(20)
			})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for initial scheduled backup")
			f.EventuallySuccessfulBackupCount(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration).Should(BeNumerically("==", 1))

			By("Pausing scheduled backup")
			backupConfig, _, err = v1beta1_util.PatchBackupConfiguration(context.TODO(), f.StashClient.StashV1beta1(), backupConfig, func(in *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
				in.Spec.Paused = true
				return in
			}, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the CronJob has been suspended")
			f.EventuallyCronJobSuspended(backupConfig.ObjectMeta).Should(BeTrue())

			By("Waiting for running backup to complete")
			f.EventuallyRunningBackupCompleted(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration).Should(BeTrue())
			initialBackupCount, err := f.GetSuccessfulBackupSessionCount(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration)
			Expect(err).NotTo(HaveOccurred())

			By("Taking instant backup")
			// Take an Instant Backup of the Sample Data
			backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta, v1beta1.BackupInvokerRef{
				Name: backupConfig.Name,
				Kind: v1beta1.ResourceKindBackupConfiguration,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the instant backup has succeeded")
			completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(context.TODO(), backupSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

			By("Resuming scheduled backup")
			backupConfig, _, err = v1beta1_util.PatchBackupConfiguration(context.TODO(), f.StashClient.StashV1beta1(), backupConfig, func(in *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
				in.Spec.Paused = false
				return in
			}, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the CronJob has been resumed")
			f.EventuallyCronJobResumed(backupConfig.ObjectMeta).Should(BeTrue())

			By("Waiting for scheduled backup after resuming")
			f.EventuallySuccessfulBackupCount(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration).Should(BeNumerically("==", initialBackupCount+1))
		})
	})

	Context("Job Model", func() {
		It("should pause scheduled backup but take instant backup", func() {
			// Create new PVC
			pvc, err := f.CreateNewPVC(fmt.Sprintf("%s-%s", framework.SourceVolume, f.App()))
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
				bc.Spec.Schedule = AtEveryMinutes
				bc.Spec.BackupHistoryLimit = pointer.Int32P(20)
			})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for initial scheduled backup")
			f.EventuallySuccessfulBackupCount(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration).Should(BeNumerically("==", 1))

			By("Pausing scheduled backup")
			backupConfig, _, err = v1beta1_util.PatchBackupConfiguration(context.TODO(), f.StashClient.StashV1beta1(), backupConfig, func(in *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
				in.Spec.Paused = true
				return in
			}, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the CronJob has been suspended")
			f.EventuallyCronJobSuspended(backupConfig.ObjectMeta).Should(BeTrue())

			By("Waiting for running backup to complete")
			f.EventuallyRunningBackupCompleted(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration).Should(BeTrue())
			initialBackupCount, err := f.GetSuccessfulBackupSessionCount(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration)
			Expect(err).NotTo(HaveOccurred())

			By("Taking instant backup")
			// Take an Instant Backup of the Sample Data
			backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta, v1beta1.BackupInvokerRef{
				Name: backupConfig.Name,
				Kind: v1beta1.ResourceKindBackupConfiguration,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the instant backup has succeeded")
			completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(context.TODO(), backupSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

			By("Resuming scheduled backup")
			backupConfig, _, err = v1beta1_util.PatchBackupConfiguration(context.TODO(), f.StashClient.StashV1beta1(), backupConfig, func(in *v1beta1.BackupConfiguration) *v1beta1.BackupConfiguration {
				in.Spec.Paused = false
				return in
			}, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the CronJob has been resumed")
			f.EventuallyCronJobResumed(backupConfig.ObjectMeta).Should(BeTrue())

			By("Waiting for scheduled backup after resuming")
			f.EventuallySuccessfulBackupCount(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration).Should(BeNumerically("==", initialBackupCount+1))
		})
	})
})
