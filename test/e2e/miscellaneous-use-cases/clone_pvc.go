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
package miscellaneous_use_cases

import (
	"fmt"
	"time"

	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Restore PVC according to VolumeClaimTemplate
				By("Restoring the backed up data using VolumeClaimTemplate")
				restoreSession, err := f.SetupRestoreProcess(metav1.ObjectMeta{}, repo, apis.KindPersistentVolumeClaim, framework.RestoredVolume, func(restore *v1beta1.RestoreSession) {
					restore.Spec.Target.VolumeClaimTemplates = []ofst.PersistentVolumeClaim{
						{
							PartialObjectMeta: ofst.PartialObjectMeta{
								Name:      framework.RestoredVolume,
								Namespace: f.Namespace(),
								CreationTimestamp: metav1.Time{
									Time: time.Now(),
								},
							},
							Spec: core.PersistentVolumeClaimSpec{
								AccessModes: []core.PersistentVolumeAccessMode{
									core.ReadWriteOnce,
								},
								StorageClassName: types.StringP(f.StorageClass),
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
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

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

				// Take an Instant Backup the Sample Data
				backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that BackupSession has succeeded")
				completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(backupSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

				// Restore PVC according to VolumeClaimTemplate
				By("Restoring the backed up data into PVC")
				restoredPVCNamePrefix := fmt.Sprintf("%s-%s-%s", framework.RestoredVolume, framework.RestoredStatefulSet, f.App())
				restoreSession, err := f.SetupRestoreProcess(metav1.ObjectMeta{}, repo, apis.KindPersistentVolumeClaim, restoredPVCNamePrefix, func(restore *v1beta1.RestoreSession) {
					restore.Spec.Target.Replicas = types.Int32P(3)
					restore.Spec.Target.VolumeClaimTemplates = []ofst.PersistentVolumeClaim{
						{
							PartialObjectMeta: ofst.PartialObjectMeta{
								Name:      fmt.Sprintf("%s-${POD_ORDINAL}", restoredPVCNamePrefix),
								Namespace: f.Namespace(),
								CreationTimestamp: metav1.Time{
									Time: time.Now(),
								},
							},
							Spec: core.PersistentVolumeClaimSpec{
								AccessModes: []core.PersistentVolumeAccessMode{
									core.ReadWriteOnce,
								},
								StorageClassName: types.StringP(f.StorageClass),
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
				completedRS, err := f.StashClient.StashV1beta1().RestoreSessions(restoreSession.Namespace).Get(restoreSession.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(completedRS.Status.Phase).Should(Equal(v1beta1.RestoreSessionSucceeded))

				//Deploy restored StatefulSet
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
