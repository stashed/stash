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
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gomodules.xyz/pointer"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

var _ = Describe("Clone", func() {
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

	Context("Deployment's Volumes", func() {
		Context("Restore using VolumeClaimTemplate", func() {
			It("should create PVC according to the VolumeClaimTemplate and restore there", func() {
				// Deploy a Deployment
				deploy, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(deploy.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(deploy.ObjectMeta, repo, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup of the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta, v1beta1.BackupInvokerRef{
					Name: backupConfig.Name,
					Kind: v1beta1.ResourceKindBackupConfiguration,
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(context.TODO(), backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Restore PVC according to VolumeClaimTemplate
				By("Restoring the backed up data using VolumeClaimTemplate")
				restoredPVCName := fmt.Sprintf("%s-%s", framework.RestoredVolume, f.App())
				restoreSession, err := f.SetupRestoreProcess(metav1.ObjectMeta{}, repo, apis.KindPersistentVolumeClaim, restoredPVCName, func(restore *v1beta1.RestoreSession) {
					restore.Spec.Target.VolumeClaimTemplates = []ofst.PersistentVolumeClaim{
						{
							PartialObjectMeta: ofst.PartialObjectMeta{
								Name:      restoredPVCName,
								Namespace: f.Namespace(),
							},
							Spec: core.PersistentVolumeClaimSpec{
								AccessModes: []core.PersistentVolumeAccessMode{
									core.ReadWriteOnce,
								},
								StorageClassName: pointer.StringP(f.StorageClass),
								Resources: core.ResourceRequirements{
									Requests: core.ResourceList{
										core.ResourceStorage: resource.MustParse("10Mi"),
									},
								},
							},
						},
					}
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

				// Deploy restored Deployment
				restoredDeploy, err := f.DeployDeployment(framework.RestoredDeployment, int32(1), framework.RestoredVolume)
				Expect(err).NotTo(HaveOccurred())

				// Get restored data
				restoredData := f.RestoredData(restoredDeploy.ObjectMeta, apis.KindDeployment)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})
	})

	Context("StatefulSet's Volumes", func() {
		AfterEach(func() {
			// StatefulSet's PVCs are not get cleanup by the CleanupTestResources() function.
			// Hence, we need to cleanup them manually.
			f.CleanupUndeletedPVCs()
		})
		Context("Restore using VolumeClaimTemplate", func() {
			It("should create PVC according to the VolumeClaimTemplate and restore there", func() {
				// Deploy a StatefulSet
				ss, err := f.DeployStatefulSet(framework.SourceStatefulSet, 3, framework.SourceVolume)
				Expect(err).NotTo(HaveOccurred())

				// Generate Sample Data
				sampleData, err := f.GenerateSampleData(ss.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Setup a Minio Repository
				repo, err := f.SetupMinioRepository()
				Expect(err).NotTo(HaveOccurred())
				f.AppendToCleanupList(repo)

				// Setup workload Backup
				backupConfig, err := f.SetupWorkloadBackup(ss.ObjectMeta, repo, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				// Take an Instant Backup of the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta, v1beta1.BackupInvokerRef{
					Name: backupConfig.Name,
					Kind: v1beta1.ResourceKindBackupConfiguration,
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(context.TODO(), backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Restore PVC according to VolumeClaimTemplate
				By("Restoring the backed up data into PVC")
				restoredPVCNamePrefix := fmt.Sprintf("%s-%s-%s", framework.RestoredVolume, framework.RestoredStatefulSet, f.App())
				restoreSession, err := f.SetupRestoreProcess(metav1.ObjectMeta{}, repo, apis.KindPersistentVolumeClaim, restoredPVCNamePrefix, func(restore *v1beta1.RestoreSession) {
					restore.Spec.Target.Replicas = pointer.Int32P(3)
					restore.Spec.Target.VolumeClaimTemplates = []ofst.PersistentVolumeClaim{
						{
							PartialObjectMeta: ofst.PartialObjectMeta{
								Name:      fmt.Sprintf("%s-${POD_ORDINAL}", restoredPVCNamePrefix),
								Namespace: f.Namespace(),
							},
							Spec: core.PersistentVolumeClaimSpec{
								AccessModes: []core.PersistentVolumeAccessMode{
									core.ReadWriteOnce,
								},
								StorageClassName: pointer.StringP(f.StorageClass),
								Resources: core.ResourceRequirements{
									Requests: core.ResourceList{
										core.ResourceStorage: resource.MustParse("10Mi"),
									},
								},
							},
						},
					}
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that RestoreSession succeeded")
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(context.TODO(), restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSucceeded))

				// Deploy restored StatefulSet
				restoredSS, err := f.DeployStatefulSet(framework.RestoredStatefulSet, int32(3), framework.RestoredVolume)
				Expect(err).NotTo(HaveOccurred())

				// Get restored data
				restoredData := f.RestoredData(restoredSS.ObjectMeta, apis.KindStatefulSet)

				// Verify that restored data is same as the original data
				By("Verifying restored data is same as the original data")
				Expect(restoredData).Should(BeSameAs(sampleData))
			})
		})
	})
})
