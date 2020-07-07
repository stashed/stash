/*
Copyright AppsCode Inc. and Contributors

Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package misc

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	v1beta1_util "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1/util"
	"stash.appscode.dev/stash/test/e2e/framework"

	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pfutil "kmodules.xyz/client-go/tools/portforward"
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
				bc.Spec.BackupHistoryLimit = types.Int32P(20)
			})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for initial scheduled backup")
			f.EventuallyBackupCount(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration).Should(BeNumerically("==", 1))

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
			f.EventuallyBackupCount(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration).Should(BeNumerically("==", initialBackupCount+1))
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
				bc.Spec.BackupHistoryLimit = types.Int32P(20)
			})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for initial scheduled backup")
			f.EventuallyBackupCount(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration).Should(BeNumerically("==", 1))

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
			f.EventuallyBackupCount(backupConfig.ObjectMeta, v1beta1.ResourceKindBackupConfiguration).Should(BeNumerically("==", initialBackupCount+1))
		})
	})

	Context("Batch Backup", func() {
		const sampleTable = "stashDemo"

		It("should pause scheduled backup but take instant backup", func() {
			var members []v1beta1.BackupConfigurationTemplateSpec

			// Deploy a Deployment and generate sample data inside Deployment
			dpl, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
			Expect(err).NotTo(HaveOccurred())
			_, err = f.GenerateSampleData(dpl.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())
			members = append(members, v1beta1.BackupConfigurationTemplateSpec{
				Task: v1beta1.TaskRef{},
				Target: &v1beta1.BackupTarget{
					Ref: v1beta1.TargetRef{
						APIVersion: apps.SchemeGroupVersion.String(),
						Kind:       apis.KindDeployment,
						Name:       dpl.Name,
					},
					Paths: []string{
						framework.TestSourceDataMountPath,
					},
					VolumeMounts: []core.VolumeMount{
						{
							Name:      framework.SourceVolume,
							MountPath: framework.TestSourceDataMountPath,
						},
					},
				},
			})

			// Create new PVC and deploy a Pod to use this pvc
			// then generate sample data inside PVC
			pvc, err := f.CreateNewPVC(fmt.Sprintf("%s-%s", framework.SourcePVC, f.App()))
			Expect(err).NotTo(HaveOccurred())
			pod, err := f.DeployPod(pvc.Name)
			Expect(err).NotTo(HaveOccurred())
			_, err = f.GenerateSampleData(pod.ObjectMeta, apis.KindPod)
			Expect(err).NotTo(HaveOccurred())
			members = append(members, v1beta1.BackupConfigurationTemplateSpec{
				Task: v1beta1.TaskRef{
					Name: framework.TaskPVCBackup,
				},
				Target: &v1beta1.BackupTarget{
					Ref: v1beta1.TargetRef{
						APIVersion: core.SchemeGroupVersion.String(),
						Kind:       apis.KindPersistentVolumeClaim,
						Name:       pvc.Name,
					},
				},
			})

			// Setup MySQL Database and generate sample data
			// Deploy MySQL database and respective service,secret,PVC and AppBinding.
			By("Deploying MySQL Server")
			dpl, appBinding, err := f.DeployMySQLDatabase()
			Expect(err).NotTo(HaveOccurred())

			By("Port forwarding MySQL pod")
			pod, err = f.GetPod(dpl.ObjectMeta)
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
			Expect(err).NotTo(HaveOccurred())
			members = append(members, v1beta1.BackupConfigurationTemplateSpec{
				Task: v1beta1.TaskRef{
					Name: framework.MySQLBackupTask,
				},
				Target: &v1beta1.BackupTarget{
					Ref: v1beta1.TargetRef{
						APIVersion: core.SchemeGroupVersion.String(),
						Kind:       apis.KindAppBinding,
						Name:       appBinding.Name,
					},
				},
			})

			// Setup a Minio Repository
			repo, err := f.SetupMinioRepository()
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(repo)

			// Setup batch backup for all targets
			backupBatch, err := f.SetupBatchBackup(repo, func(in *v1beta1.BackupBatch) {
				in.Spec.Members = members
				in.Spec.Schedule = "* * * * *"
				in.Spec.BackupHistoryLimit = types.Int32P(20)
			})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for initial scheduled backup")
			f.EventuallyBackupCount(backupBatch.ObjectMeta, v1beta1.ResourceKindBackupBatch).Should(BeNumerically("==", 1))

			By("Pausing scheduled backup")
			backupBatch, _, err = v1beta1_util.PatchBackupBatch(context.TODO(), f.StashClient.StashV1beta1(), backupBatch, func(in *v1beta1.BackupBatch) *v1beta1.BackupBatch {
				in.Spec.Paused = true
				return in
			}, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the CronJob has been suspended")
			f.EventuallyCronJobSuspended(backupBatch.ObjectMeta).Should(BeTrue())

			By("Waiting for running backup to complete")
			f.EventuallyRunningBackupCompleted(backupBatch.ObjectMeta, v1beta1.ResourceKindBackupBatch).Should(BeTrue())
			initialBackupCount, err := f.GetSuccessfulBackupSessionCount(backupBatch.ObjectMeta, v1beta1.ResourceKindBackupBatch)
			Expect(err).NotTo(HaveOccurred())

			By("Taking instant backup")
			// Take an Instant Backup of the Sample Data
			backupSession, err := f.TakeInstantBackup(backupBatch.ObjectMeta, v1beta1.BackupInvokerRef{
				Name: backupBatch.Name,
				Kind: v1beta1.ResourceKindBackupBatch,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the instant backup has succeeded")
			completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(context.TODO(), backupSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

			By("Resuming scheduled backup")
			backupBatch, _, err = v1beta1_util.PatchBackupBatch(context.TODO(), f.StashClient.StashV1beta1(), backupBatch, func(in *v1beta1.BackupBatch) *v1beta1.BackupBatch {
				in.Spec.Paused = false
				return in
			}, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the CronJob has been resumed")
			f.EventuallyCronJobResumed(backupBatch.ObjectMeta).Should(BeTrue())

			By("Waiting for scheduled backup after resuming")
			f.EventuallyBackupCount(backupBatch.ObjectMeta, v1beta1.ResourceKindBackupBatch).Should(BeNumerically("==", initialBackupCount+1))
		})
	})
})
