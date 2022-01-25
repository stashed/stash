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
	"net"
	"path/filepath"

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

var _ = Describe("Snapshot Tests", func() {
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

	type testResources struct {
		workloadMeta  metav1.ObjectMeta
		workload      interface{}
		workloadKind  string
		repository    *v1alpha1.Repository
		backupConfig  *v1beta1.BackupConfiguration
		backupSession *v1beta1.BackupSession
	}

	var (
		deployWorkloads = func(securityContext *core.SecurityContext) []testResources {
			var resources []testResources
			// Deploy a Deployment
			deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume, func(dp *apps.Deployment) {
				dp.Spec.Template.Spec.Containers[0].SecurityContext = securityContext
			})
			Expect(err).NotTo(HaveOccurred())

			// Generate Sample Data
			_, err = f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())
			resources = append(resources, testResources{workloadMeta: deployment.ObjectMeta, workloadKind: apis.KindDeployment, workload: deployment})

			//// Deploy a StatefulSet
			//ss, err := f.DeployStatefulSet(framework.SourceStatefulSet, int32(3), framework.SourceVolume, func(ss *apps.StatefulSet) {
			//	ss.Spec.Template.Spec.Containers[0].SecurityContext = securityContext
			//})
			//Expect(err).NotTo(HaveOccurred())
			//
			//// Generate Sample Data
			//_, err = f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)
			//Expect(err).NotTo(HaveOccurred())
			//resources = append(resources, testResources{workloadMeta: ss.ObjectMeta, workloadKind: apis.KindStatefulSet, workload: ss})

			// Deploy a DaemonSet
			dmn, err := f.DeployDaemonSet(framework.SourceDaemonSet, framework.SourceVolume, func(dmn *apps.DaemonSet) {
				dmn.Spec.Template.Spec.Containers[0].SecurityContext = securityContext
			})
			Expect(err).NotTo(HaveOccurred())

			// Generate Sample Data
			_, err = f.GenerateSampleData(dmn.ObjectMeta, apis.KindDaemonSet)
			Expect(err).NotTo(HaveOccurred())
			resources = append(resources, testResources{workloadMeta: dmn.ObjectMeta, workloadKind: apis.KindDaemonSet, workload: dmn})

			return resources
		}

		setupBackup = func(res testResources, containerRuntimeSettings *ofst.ContainerRuntimeSettings) *v1beta1.BackupConfiguration {
			backupConfig := f.GetBackupConfiguration(res.repository.Name, func(bc *v1beta1.BackupConfiguration) {
				bc.Spec.Target = &v1beta1.BackupTarget{
					Alias: res.workloadMeta.Name,
					Ref:   framework.GetTargetRef(res.workloadMeta.Name, res.workloadKind),
					Paths: []string{
						framework.TestSourceDataMountPath,
					},
					VolumeMounts: []core.VolumeMount{
						{
							Name:      framework.SourceVolume,
							MountPath: framework.TestSourceDataMountPath,
						},
					},
				}
				bc.Spec.RuntimeSettings.Container = containerRuntimeSettings
			})
			err := f.CreateBackupConfiguration(*backupConfig)
			Expect(err).NotTo(HaveOccurred())
			f.AppendToCleanupList(backupConfig)
			return backupConfig
		}

		performOperationOnSnapshot = func(repoName, hostname string) {
			By("Listing all snapshots")
			snapshots, err := f.ListSnapshots("")
			Expect(err).NotTo(HaveOccurred())

			Expect(len(snapshots.Items)).ShouldNot(BeZero())

			By("Get a particular snapshot")
			snapshots, err = f.ListSnapshots("")
			Expect(err).NotTo(HaveOccurred())
			singleSnapshot, err := f.GetSnapshot(snapshots.Items[len(snapshots.Items)-1].Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(singleSnapshot.Name).To(BeEquivalentTo(snapshots.Items[len(snapshots.Items)-1].Name))

			By("Filter by repository name")
			snapshots, err = f.ListSnapshots("repository=" + repoName)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(snapshots.Items)).ShouldNot(BeZero())
			Expect(snapshots).Should(ComeFrom(repoName))

			By("Filter by hostname")
			snapshots, err = f.ListSnapshots("hostname=" + hostname)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(snapshots.Items)).ShouldNot(BeZero())
			Expect(snapshots).Should(HaveHostname(hostname))

			By("Filter by negated selector")
			snapshots, err = f.ListSnapshots("repository!=" + repoName)
			Expect(err).NotTo(HaveOccurred())
			if len(snapshots.Items) > 0 {
				Expect(snapshots).ShouldNot(ComeFrom(repoName))
			}

			By("Filter by set based selector")
			snapshots, err = f.ListSnapshots("repository in(" + repoName + ")")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(snapshots.Items)).ShouldNot(BeZero())
			Expect(snapshots).Should(ComeFrom(repoName))

			snapshots, err = f.ListSnapshots("repository=" + repoName)
			Expect(err).NotTo(HaveOccurred())
			snapshotToDelete := snapshots.Items[len(snapshots.Items)-1].Name
			By("Deleting snapshot " + snapshotToDelete)
			err = f.DeleteSnapshot(snapshotToDelete)
			Expect(err).NotTo(HaveOccurred())

			By("Checking deleted snapshot not exist")
			_, err = f.GetSnapshot(snapshotToDelete)
			Expect(err).To(HaveOccurred())
		}
	)

	Context("Backend", func() {

		Context("Minio", func() {
			It("should successfully perform Snapshot operations", func() {
				// Deploy the workloads
				resources := deployWorkloads(nil)
				// Create Repositories
				for i := range resources {
					repo, err := f.SetupMinioRepository(func(repo *v1alpha1.Repository) {
						repo.Spec.Backend.S3.Prefix = filepath.Join(repo.Spec.Backend.S3.Prefix, resources[i].workloadMeta.Name)
					})
					Expect(err).NotTo(HaveOccurred())
					resources[i].repository = repo
				}
				By("Creating BackupConfiguration for the workloads")
				for i := range resources {
					backupConfig := setupBackup(resources[i], nil)
					resources[i].backupConfig = backupConfig
				}
				// Wait for the workloads to be ready with sidecar
				for i := range resources {
					err := f.WaitForTargetToBeReadyWithSidecar(resources[i].workloadKind, resources[i].workloadMeta)
					Expect(err).NotTo(HaveOccurred())
				}
				By("Triggering an instant backup for the workloads")
				for i := range resources {
					backupSession, err := f.TriggerInstantBackup(resources[i].workloadMeta, v1beta1.BackupInvokerRef{
						Kind: v1beta1.ResourceKindBackupConfiguration,
						Name: resources[i].backupConfig.Name,
					})
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(backupSession)
					resources[i].backupSession = backupSession
				}
				By("Waiting for the backup processes to complete")
				for i := range resources {
					f.EventuallyBackupProcessCompleted(resources[i].backupSession.ObjectMeta).Should(BeTrue())
				}
				By("Verifying that the backup process has succeeded for the workloads")
				for i := range resources {
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(resources[i].backupSession.Namespace).Get(context.TODO(), resources[i].backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
				}
				// Perform snapshots operation
				performOperationOnSnapshot(resources[0].repository.Name, resources[0].backupConfig.Spec.Target.Alias)
			})
		})

		Context("Azure", func() {
			It("should successfully perform Snapshot operations", func() {
				// Deploy the workloads
				resources := deployWorkloads(nil)
				// Create Repositories
				for i := range resources {
					repo, err := f.SetupAzureRepository(0, true, func(repo *v1alpha1.Repository) {
						repo.Spec.Backend.Azure.Prefix = filepath.Join(repo.Spec.Backend.Azure.Prefix, resources[i].workloadMeta.Name)
					})
					Expect(err).NotTo(HaveOccurred())
					resources[i].repository = repo
				}
				By("Creating BackupConfiguration for the workloads")
				for i := range resources {
					backupConfig := setupBackup(resources[i], nil)
					resources[i].backupConfig = backupConfig
				}
				// Wait for the workloads to be ready with sidecar
				for i := range resources {
					err := f.WaitForTargetToBeReadyWithSidecar(resources[i].workloadKind, resources[i].workloadMeta)
					Expect(err).NotTo(HaveOccurred())
				}
				By("Triggering an instant backup for the workloads")
				for i := range resources {
					backupSession, err := f.TriggerInstantBackup(resources[i].workloadMeta, v1beta1.BackupInvokerRef{
						Kind: v1beta1.ResourceKindBackupConfiguration,
						Name: resources[i].backupConfig.Name,
					})
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(backupSession)
					resources[i].backupSession = backupSession
				}
				By("Waiting for the backup processes to complete")
				for i := range resources {
					f.EventuallyBackupProcessCompleted(resources[i].backupSession.ObjectMeta).Should(BeTrue())
				}
				By("Verifying that the backup process has succeeded for the workloads")
				for i := range resources {
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(resources[i].backupSession.Namespace).Get(context.TODO(), resources[i].backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
				}
				// Perform snapshots operation
				performOperationOnSnapshot(resources[0].repository.Name, resources[0].backupConfig.Spec.Target.Alias)
			})
		})

		Context("GCS", func() {
			It("should successfully perform Snapshot operations", func() {
				// Deploy the workloads
				resources := deployWorkloads(nil)
				// Create Repositories
				for i := range resources {
					repo, err := f.SetupGCSRepository(0, true, func(repo *v1alpha1.Repository) {
						repo.Spec.Backend.GCS.Prefix = filepath.Join(repo.Spec.Backend.GCS.Prefix, resources[i].workloadMeta.Name)
					})
					Expect(err).NotTo(HaveOccurred())
					resources[i].repository = repo
				}
				By("Creating BackupConfiguration for the workloads")
				for i := range resources {
					backupConfig := setupBackup(resources[i], nil)
					resources[i].backupConfig = backupConfig
				}
				// Wait for the workloads to be ready with sidecar
				for i := range resources {
					err := f.WaitForTargetToBeReadyWithSidecar(resources[i].workloadKind, resources[i].workloadMeta)
					Expect(err).NotTo(HaveOccurred())
				}
				By("Triggering an instant backup for the workloads")
				for i := range resources {
					backupSession, err := f.TriggerInstantBackup(resources[i].workloadMeta, v1beta1.BackupInvokerRef{
						Kind: v1beta1.ResourceKindBackupConfiguration,
						Name: resources[i].backupConfig.Name,
					})
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(backupSession)
					resources[i].backupSession = backupSession
				}
				By("Waiting for the backup processes to complete")
				for i := range resources {
					f.EventuallyBackupProcessCompleted(resources[i].backupSession.ObjectMeta).Should(BeTrue())
				}
				By("Verifying that the backup process has succeeded for the workloads")
				for i := range resources {
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(resources[i].backupSession.Namespace).Get(context.TODO(), resources[i].backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
				}
				// Perform snapshots operation
				performOperationOnSnapshot(resources[0].repository.Name, resources[0].backupConfig.Spec.Target.Alias)
			})
		})

		Context("S3", func() {
			It("should successfully perform Snapshot operations", func() {
				// Deploy the workloads
				resources := deployWorkloads(nil)
				// Create Repositories
				for i := range resources {
					repo, err := f.SetupS3Repository(true, func(repo *v1alpha1.Repository) {
						repo.Spec.Backend.S3.Prefix = filepath.Join(repo.Spec.Backend.S3.Prefix, resources[i].workloadMeta.Name)
					})
					Expect(err).NotTo(HaveOccurred())
					resources[i].repository = repo
				}
				By("Creating BackupConfiguration for the workloads")
				for i := range resources {
					backupConfig := setupBackup(resources[i], nil)
					resources[i].backupConfig = backupConfig
				}
				// Wait for the workloads to be ready with sidecar
				for i := range resources {
					err := f.WaitForTargetToBeReadyWithSidecar(resources[i].workloadKind, resources[i].workloadMeta)
					Expect(err).NotTo(HaveOccurred())
				}
				By("Triggering an instant backup for the workloads")
				for i := range resources {
					backupSession, err := f.TriggerInstantBackup(resources[i].workloadMeta, v1beta1.BackupInvokerRef{
						Kind: v1beta1.ResourceKindBackupConfiguration,
						Name: resources[i].backupConfig.Name,
					})
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(backupSession)
					resources[i].backupSession = backupSession
				}
				By("Waiting for the backup processes to complete")
				for i := range resources {
					f.EventuallyBackupProcessCompleted(resources[i].backupSession.ObjectMeta).Should(BeTrue())
				}
				By("Verifying that the backup process has succeeded for the workloads")
				for i := range resources {
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(resources[i].backupSession.Namespace).Get(context.TODO(), resources[i].backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
				}
				// Perform snapshots operation
				performOperationOnSnapshot(resources[0].repository.Name, resources[0].backupConfig.Spec.Target.Alias)
			})
		})

		Context("Swift", func() {
			It("should successfully perform Snapshot operations", func() {
				// Deploy the workloads
				resources := deployWorkloads(nil)
				// Create Repositories
				for i := range resources {
					repo, err := f.SetupSwiftRepository(true, func(repo *v1alpha1.Repository) {
						repo.Spec.Backend.Swift.Prefix = filepath.Join(repo.Spec.Backend.Swift.Prefix, resources[i].workloadMeta.Name)
					})
					Expect(err).NotTo(HaveOccurred())
					resources[i].repository = repo
				}
				By("Creating BackupConfiguration for the workloads")
				for i := range resources {
					backupConfig := setupBackup(resources[i], nil)
					resources[i].backupConfig = backupConfig
				}
				// Wait for the workloads to be ready with sidecar
				for i := range resources {
					err := f.WaitForTargetToBeReadyWithSidecar(resources[i].workloadKind, resources[i].workloadMeta)
					Expect(err).NotTo(HaveOccurred())
				}
				By("Triggering an instant backup for the workloads")
				for i := range resources {
					backupSession, err := f.TriggerInstantBackup(resources[i].workloadMeta, v1beta1.BackupInvokerRef{
						Kind: v1beta1.ResourceKindBackupConfiguration,
						Name: resources[i].backupConfig.Name,
					})
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(backupSession)
					resources[i].backupSession = backupSession
				}
				By("Waiting for the backup processes to complete")
				for i := range resources {
					f.EventuallyBackupProcessCompleted(resources[i].backupSession.ObjectMeta).Should(BeTrue())
				}
				By("Verifying that the backup process has succeeded for the workloads")
				for i := range resources {
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(resources[i].backupSession.Namespace).Get(context.TODO(), resources[i].backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
				}
				// Perform snapshots operation
				performOperationOnSnapshot(resources[0].repository.Name, resources[0].backupConfig.Spec.Target.Alias)
			})
		})

		Context("B2", func() {
			It("should successfully perform Snapshot operations", func() {
				// Deploy the workloads
				resources := deployWorkloads(nil)
				// Create Repositories
				for i := range resources {
					repo, err := f.SetupB2Repository(0, func(repo *v1alpha1.Repository) {
						repo.Spec.Backend.B2.Prefix = filepath.Join(repo.Spec.Backend.B2.Prefix, resources[i].workloadMeta.Name)
					})
					Expect(err).NotTo(HaveOccurred())
					resources[i].repository = repo
				}
				By("Creating BackupConfiguration for the workloads")
				for i := range resources {
					backupConfig := setupBackup(resources[i], nil)
					resources[i].backupConfig = backupConfig
				}
				// Wait for the workloads to be ready with sidecar
				for i := range resources {
					err := f.WaitForTargetToBeReadyWithSidecar(resources[i].workloadKind, resources[i].workloadMeta)
					Expect(err).NotTo(HaveOccurred())
				}
				By("Triggering an instant backup for the workloads")
				for i := range resources {
					backupSession, err := f.TriggerInstantBackup(resources[i].workloadMeta, v1beta1.BackupInvokerRef{
						Kind: v1beta1.ResourceKindBackupConfiguration,
						Name: resources[i].backupConfig.Name,
					})
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(backupSession)
					resources[i].backupSession = backupSession
				}
				By("Waiting for the backup processes to complete")
				for i := range resources {
					f.EventuallyBackupProcessCompleted(resources[i].backupSession.ObjectMeta).Should(BeTrue())
				}
				By("Verifying that the backup process has succeeded for the workloads")
				for i := range resources {
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(resources[i].backupSession.Namespace).Get(context.TODO(), resources[i].backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
				}
				// Perform snapshots operation
				performOperationOnSnapshot(resources[0].repository.Name, resources[0].backupConfig.Spec.Target.Alias)
			})
		})

		Context("Rest Server", func() {
			BeforeEach(func() {
				By("Creating Rest Server")
				_, err := f.CreateRestServer(true, []net.IP{net.ParseIP("127.0.0.1")})
				Expect(err).NotTo(HaveOccurred())
			})
			It("should successfully perform Snapshot operations", func() {
				// Deploy the workloads
				resources := deployWorkloads(nil)

				By("Creating user in the rest server for the individual workload")
				for i := range resources {
					err := f.CreateRestUser(resources[i].workloadMeta.Name)
					Expect(err).NotTo(HaveOccurred())
				}
				// Create Repositories
				for i := range resources {
					repo, err := f.SetupRestRepository(true, resources[i].workloadMeta.Name, framework.TEST_REST_SERVER_PASSWORD, func(repo *v1alpha1.Repository) {
						repo.Spec.Backend.Rest.URL = fmt.Sprintf("https://%s:8000/%s", f.RestServiceAddres(), resources[i].workloadMeta.Name)
					})
					Expect(err).NotTo(HaveOccurred())
					resources[i].repository = repo
				}
				By("Creating BackupConfiguration for the workloads")
				for i := range resources {
					backupConfig := setupBackup(resources[i], nil)
					resources[i].backupConfig = backupConfig
				}
				// Wait for the workloads to be ready with sidecar
				for i := range resources {
					err := f.WaitForTargetToBeReadyWithSidecar(resources[i].workloadKind, resources[i].workloadMeta)
					Expect(err).NotTo(HaveOccurred())
				}
				By("Triggering an instant backup for the workloads")
				for i := range resources {
					backupSession, err := f.TriggerInstantBackup(resources[i].workloadMeta, v1beta1.BackupInvokerRef{
						Kind: v1beta1.ResourceKindBackupConfiguration,
						Name: resources[i].backupConfig.Name,
					})
					Expect(err).NotTo(HaveOccurred())
					f.AppendToCleanupList(backupSession)
					resources[i].backupSession = backupSession
				}
				By("Waiting for the backup processes to complete")
				for i := range resources {
					f.EventuallyBackupProcessCompleted(resources[i].backupSession.ObjectMeta).Should(BeTrue())
				}
				By("Verifying that the backup process has succeeded for the workloads")
				for i := range resources {
					completedBS, err := f.StashClient.StashV1beta1().BackupSessions(resources[i].backupSession.Namespace).Get(context.TODO(), resources[i].backupSession.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))
				}
				// Perform snapshots operation
				performOperationOnSnapshot(resources[0].repository.Name, resources[0].backupConfig.Spec.Target.Alias)
			})
		})
	})

})
