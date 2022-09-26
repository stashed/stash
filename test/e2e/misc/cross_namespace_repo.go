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

	"stash.appscode.dev/apimachinery/apis"
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/invoker"
	"stash.appscode.dev/stash/test/e2e/framework"
	. "stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	allowSameNamespace = v1alpha1.NamespacesFromSame
	allowAllNamespaces = v1alpha1.NamespacesFromAll
)

var _ = Describe("Cross Namespace Repository", func() {
	var f *framework.Invocation

	BeforeEach(func() {
		f = framework.NewInvocation()

		By("Creating backup namespace: " + f.BackupNamespace())
		err := f.CreateNamespace(f.NewNamespace(f.BackupNamespace()))
		Expect(err).NotTo(HaveOccurred())

		By("Create restore namespace: " + f.RestoreNamespace())
		err = f.CreateNamespace(f.NewNamespace(f.RestoreNamespace()))
		Expect(err).NotTo(HaveOccurred())
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

		By("Deleting namespace: " + f.BackupNamespace())
		err = f.DeleteNamespace(f.BackupNamespace())
		Expect(err).NotTo(HaveOccurred())

		By("Deleting namespace: " + f.RestoreNamespace())
		err = f.DeleteNamespace(f.RestoreNamespace())
		Expect(err).NotTo(HaveOccurred())
	})

	Context("No UsagePolicy", func() {
		It("should allow BackupConfigurations only from the same namespace", func() {
			deployment := f.Deployment(framework.SourceDeployment, framework.SourcePVC, framework.SourceVolume)

			repo, err := f.SetupMinioRepository()
			Expect(err).NotTo(HaveOccurred())

			_, err = f.CreateBackupConfigForWorkload(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
				bc.Namespace = repo.Namespace
				bc.Spec.Repository.Namespace = f.Namespace()
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject BackupConfiguration from different namespaces", func() {
			deployment := f.Deployment(framework.SourceDeployment, framework.SourcePVC, framework.SourceVolume)

			repo, err := f.SetupMinioRepository()
			Expect(err).NotTo(HaveOccurred())

			_, err = f.CreateBackupConfigForWorkload(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
				bc.Namespace = f.BackupNamespace()
				bc.Spec.Repository.Namespace = f.Namespace()
			})
			Expect(err).To(HaveOccurred())
		})

		It("should allow RestoreSession from the same namespace", func() {
			deployment := f.Deployment(framework.SourceDeployment, framework.SourcePVC, framework.SourceVolume)

			repo, err := f.SetupMinioRepository()
			Expect(err).NotTo(HaveOccurred())

			_, err = f.CreateRestoreSessionForWorkload(deployment.ObjectMeta, repo.Name, apis.KindDeployment, framework.SourceVolume, func(rs *v1beta1.RestoreSession) {
				rs.Namespace = repo.Namespace
				rs.Spec.Repository.Namespace = f.Namespace()
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject RestoreSession from different namespaces", func() {
			deployment := f.Deployment(framework.SourceDeployment, framework.SourcePVC, framework.SourceVolume)

			repo, err := f.SetupMinioRepository()
			Expect(err).NotTo(HaveOccurred())

			_, err = f.CreateRestoreSessionForWorkload(deployment.ObjectMeta, repo.Name, apis.KindDeployment, framework.SourceVolume, func(rs *v1beta1.RestoreSession) {
				rs.Namespace = f.RestoreNamespace()
				rs.Spec.Repository.Namespace = f.Namespace()
			})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Repository Created Later", func() {
		It("invoker phase should update with Repository changes", func() {
			deployment, err := f.DeployDeployment(framework.SourceDeployment, int32(1), framework.SourceVolume, func(dp *apps.Deployment) {
				dp.Namespace = f.BackupNamespace()
			})
			Expect(err).NotTo(HaveOccurred())

			sampleData, err := f.GenerateSampleData(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Storage Secret")
			cred := f.SecretForMinioBackend(true)

			if missing, _ := BeZero().Match(cred); missing {
				Skip("Missing Minio credential")
			}
			_, err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			repo := f.NewMinioRepository(cred.Name)

			backupConfig, err := f.CreateBackupConfigForWorkload(deployment.ObjectMeta, repo, apis.KindDeployment, func(bc *v1beta1.BackupConfiguration) {
				bc.Namespace = f.BackupNamespace()
				bc.Spec.Repository.Namespace = f.Namespace()
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the backup invoker phase is: NotReady")
			inv := invoker.NewBackupConfigurationInvoker(f.StashClient, backupConfig)
			f.EventuallyBackupInvokerPhase(inv).Should(BeEquivalentTo(v1beta1.BackupInvokerNotReady))

			By("Creating Repository not allowing the BackupConfiguration")
			repo.Spec.UsagePolicy = &v1alpha1.UsagePolicy{
				AllowedNamespaces: v1alpha1.AllowedNamespaces{
					From: &allowSameNamespace,
				},
			}
			_, err = f.CreateRepository(repo, &cred)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the backup invoker phase is: Invalid")
			f.EventuallyBackupInvokerPhase(inv).Should(BeEquivalentTo(v1beta1.BackupInvokerInvalid))

			By("Updating Repository to allow all namespaces")
			_, err = f.AllowNamespace(repo, allowAllNamespaces)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the backup invoker phase is: Ready")
			f.EventuallyBackupInvokerPhase(inv).Should(BeEquivalentTo(v1beta1.BackupInvokerReady))

			backupSession, err := f.TakeInstantBackup(backupConfig.ObjectMeta, v1beta1.BackupInvokerRef{
				Name: backupConfig.Name,
				Kind: v1beta1.ResourceKindBackupConfiguration,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that BackupSession has succeeded")
			completedBS, err := f.StashClient.StashV1beta1().BackupSessions(backupSession.Namespace).Get(context.TODO(), backupSession.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(completedBS.Status.Phase).Should(Equal(v1beta1.BackupSessionSucceeded))

			By("Deleting the Repository")
			err = f.DeleteRepository(repo)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that BackupConfiguration phase is: NotReady")
			f.EventuallyBackupInvokerPhase(inv).Should(BeEquivalentTo(v1beta1.BackupInvokerNotReady))

			restoredDeployment, err := f.DeployDeployment(framework.RestoredDeployment, int32(1), framework.RestoredVolume, func(dp *apps.Deployment) {
				dp.Namespace = f.RestoreNamespace()
			})
			Expect(err).NotTo(HaveOccurred())

			By("Creating RestoreSession")
			restoreSession, err := f.CreateRestoreSessionForWorkload(restoredDeployment.ObjectMeta, repo.Name, apis.KindDeployment, framework.RestoredVolume, func(restore *v1beta1.RestoreSession) {
				restore.Namespace = f.RestoreNamespace()
				restore.Spec.Repository.Namespace = f.Namespace()
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that RestoreSession phase is Pending")
			restoreInvoker := invoker.NewRestoreSessionInvoker(f.KubeClient, f.StashClient, restoreSession)
			f.EventuallyRestoreInvokerPhase(restoreInvoker).Should(BeEquivalentTo(v1beta1.RestorePending))

			By("Creating Repository not allowing the RestoreSession")
			_, err = f.CreateRepository(repo, &cred)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that RestoreSession phase is Invalid")
			f.EventuallyRestoreInvokerPhase(restoreInvoker).Should(BeEquivalentTo(v1beta1.RestorePhaseInvalid))

			By("Updating Repository to allow the RestoreSession")
			_, err = f.AllowNamespace(repo, allowAllNamespaces)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that RestoreSession phase is Succeeded")
			f.EventuallyRestoreInvokerPhase(restoreInvoker).Should(BeEquivalentTo(v1beta1.RestoreSucceeded))

			By("Verifying restored data is same as the original data")
			restoredData := f.RestoredData(restoredDeployment.ObjectMeta, apis.KindDeployment)
			Expect(restoredData).Should(BeSameAs(sampleData))
		})
	})
})
