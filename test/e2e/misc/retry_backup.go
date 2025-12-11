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
	"time"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"

	. "github.com/onsi/ginkgo/v2" // nolint: staticcheck
	. "github.com/onsi/gomega"    // nolint: staticcheck
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Retry Backup", func() {
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
	})

	Context("Deployment", func() {
		It("should retry if backup failed", func() {
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
			retryLimit := int32(3)
			historyLimit := int32(5)
			nextSchedule := (time.Now().Minute() + 1) % 60

			backupConfig, err := f.SetupWorkloadBackup(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
				bc.Spec.RetryConfig = &v1beta1.RetryConfig{
					MaxRetry: retryLimit,
					Delay:    metav1.Duration{Duration: 30 * time.Second},
				}
				bc.Spec.BackupHistoryLimit = &historyLimit
				bc.Spec.Schedule = fmt.Sprintf("*/%d * * * *", nextSchedule)

				bc.Spec.Target.Paths = []string{"/path/does/not/exist"}
			})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for next schedule backup")
			f.EventuallyBackupSessionCreated(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration).Should(BeTrue())
			sessions, err := f.GetBackupSessionsForInvoker(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(sessions.Items)).To(BeNumerically(">", 0))

			backupSession := sessions.Items[0]
			By("Waiting for BackupSession to complete")
			f.EventuallyBackupProcessCompleted(backupSession.ObjectMeta).Should(BeTrue())

			By("Verifying that BackupSession has failed")
			completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(context.TODO(), backupSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionFailed))

			By("Verifying that BackupSession has retryLeft field set")
			Expect(completedBS.Spec.RetryLeft).To(BeEquivalentTo(retryLimit))

			By(fmt.Sprintf("Waiting for backup to retry: %d times", retryLimit))
			f.EventuallyCompletedBackupSessionCount(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration).Should(BeNumerically(">=", retryLimit+1))

			sessions, err = f.GetBackupSessionsForInvoker(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration)
			Expect(err).NotTo(HaveOccurred())

			retriedBackup := 0
			for _, s := range sessions.Items {
				if s.Spec.RetryLeft > 0 {
					By("Verifying that retried field has been set for BackupSession: " + s.Name)
					Expect(*s.Status.Retried).Should(BeTrue())
					retriedBackup++

					By("Verifying that nextRetry field has been set for BackupSession: " + s.Name)
					Expect(s.Status.NextRetry).ShouldNot(BeNil())
				}
			}
			By("Verifying that the number of retried backup matches the maxRetry field")
			Expect(retriedBackup).To(BeEquivalentTo(retryLimit))
		})
	})
})
