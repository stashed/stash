package e2e_test

import (
	"github.com/appscode/go/types"
	"github.com/appscode/stash/apis"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/appscode/stash/pkg/util"
	"github.com/appscode/stash/test/e2e/framework"
	"github.com/appscode/stash/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
)

var _ = Describe("Deployment", func() {
	var (
		err              error
		f                *framework.Invocation
		cred             core.Secret
		deployment       apps.Deployment
		recoveredDeployment apps.Deployment
		repo             *api.Repository
		backupCfg        v1beta1.BackupConfiguration
		restoreSession   v1beta1.RestoreSession
		pvc              *core.PersistentVolumeClaim
		targetref		 v1beta1.TargetRef
	)
	var(
		sampleData []string
		restoredData []string
	)
	BeforeEach(func() {
		f = root.Invoke()
	})
	JustBeforeEach(func() {
		cred = f.SecretForLocalBackend()
		if missing, _ := BeZero().Match(cred); missing {
			Skip("Missing repository credential")
		}
		pvc = f.GetPersistentVolumeClaim()
		err = f.CreatePersistentVolumeClaim(pvc)
		Expect(err).NotTo(HaveOccurred())
		repo = f.Repository(cred.Name, pvc.Name)

		backupCfg = f.BackupConfiguration(repo.Name, targetref)
		restoreSession = f.RestoreSession(repo.Name, targetref)
	})
	AfterEach(func() {
		err = f.DeleteSecret(cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
		err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
	})
	var (
		shouldBackupNewDeployment = func() {
			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())
			err = util.WaitUntilDeploymentReady(f.KubeClient, deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Creating sample data inside workload")
			err = f.CreateSampleDataInsideWorkload(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			By("Reading sample data from /source/data mountPath inside workload")
			sampleData, err = f.ReadSampleDataFromFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleData).To(Not(BeEmpty()))

			By("Creating storage Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating new repository")
			err = f.CreateRepository(repo)
			Expect(err).NotTo(HaveOccurred())

			By("Creating BackupConfiguration" + backupCfg.Name)
			err = f.CreateBackupConfiguration(backupCfg)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(matcher.HaveSidecar(util.StashContainer))

			By("Waiting for BackupSession")
			f.EventuallyBackupSessionCreated(backupCfg.ObjectMeta).Should(BeTrue())
			bs, err := f.GetBackupSession(backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Check for succeeded BackupSession")
			f.EventuallyBackupSessionPhase(bs.ObjectMeta).Should(Equal(v1beta1.BackupSessionSucceeded))

			By("Check for repository status updated")
			f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Delete BackupConfiguration")
			err = f.DeleteBackupConfiguration(backupCfg)
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient,backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting to remove sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).ShouldNot(matcher.HaveSidecar(util.StashContainer))
			err = util.WaitUntilDeploymentReady(f.KubeClient, deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		}
	)
	Context("Backup && Restore for Deployment", func() {
		BeforeEach(func() {
			pvc = f.GetPersistentVolumeClaim()
			err = f.CreatePersistentVolumeClaim(pvc)
			Expect(err).NotTo(HaveOccurred())
			deployment = f.Deployment(pvc.Name)

			targetref = v1beta1.TargetRef{
				APIVersion: "apps/v1",
				Kind:       apis.KindDeployment,
				Name:       deployment.Name,
			}
		})
		AfterEach(func() {
			err := f.DeleteDeployment(deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			f.DeleteRepositories(f.DeploymentRepos(&deployment))
			err =framework.WaitUntilDeploymentDeleted(f.KubeClient,deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		})
		It("General Backup new Deployment", func() {
			shouldBackupNewDeployment()
			By("Remove sample data from workload")
			err = f.CleanupSampleDataFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())
			By("Creating Restore Session")
			err = f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for initContainer")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(matcher.HaveInitContainer(util.StashInitContainer))

			By("Waiting for restore to succeed")
			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))
			err = util.WaitUntilDeploymentReady(f.KubeClient, deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("checking the workload data has been restored")
			restoredData, err = f.ReadSampleDataFromMountedDirectory(deployment.ObjectMeta, framework.GetPathsFromRestoreSession(&restoreSession), apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(restoredData).To(Not(BeEmpty()))

			By("Verifying restored data is same as original data")
			Expect(restoredData).To(BeEquivalentTo(sampleData))

		})
	})
	Context("Leader election and backup && restore for Deployment", func() {
		BeforeEach(func() {
			pvc = f.GetPersistentVolumeClaim()
			err = f.CreatePersistentVolumeClaim(pvc)
			Expect(err).NotTo(HaveOccurred())
			deployment = f.Deployment(pvc.Name)

			targetref = v1beta1.TargetRef{
				APIVersion: "apps/v1",
				Kind:       apis.KindDeployment,
				Name:       deployment.Name,
			}
		})
		AfterEach(func() {
			err := f.DeleteDeployment(deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			f.DeleteRepositories(f.DeploymentRepos(&deployment))
			err =framework.WaitUntilDeploymentDeleted(f.KubeClient,deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})
		It("Should leader elect and Backup new deployment", func() {
			deployment.Spec.Replicas = types.Int32P(2) // two replicas
			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())
			err = util.WaitUntilDeploymentReady(f.KubeClient, deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Creating sample data inside workload")
			err = f.CreateSampleDataInsideWorkload(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			By("Creating storage Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating new repository")
			err = f.CreateRepository(repo)
			Expect(err).NotTo(HaveOccurred())

			By("Creating BackupConfiguration" + backupCfg.Name)
			err = f.CreateBackupConfiguration(backupCfg)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(matcher.HaveSidecar(util.StashContainer))

			By("Waiting for leader election")
			f.CheckLeaderElection(deployment.ObjectMeta, apis.KindDeployment, v1beta1.ResourceKindBackupConfiguration)

			By("Waiting for BackupSession")
			f.EventuallyBackupSessionCreated(backupCfg.ObjectMeta).Should(BeTrue())
			bs, err := f.GetBackupSession(backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Check for succeeded BackupSession")
			f.EventuallyBackupSessionPhase(bs.ObjectMeta).Should(Equal(v1beta1.BackupSessionSucceeded))

			By("Check for repository status updated")
			f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Delete BackupConfiguration")
			err = f.DeleteBackupConfiguration(backupCfg)
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient,backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar to be removed")
			f.EventuallyDeployment(deployment.ObjectMeta).ShouldNot(matcher.HaveSidecar(util.StashContainer))

			By("Delete sample data from workload")
			err = f.CleanupSampleDataFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Restore Session")
			err = f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for initContainer")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(matcher.HaveInitContainer(util.StashInitContainer))

			By("Waiting for restore to succeed")
			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))

		})
	})
	Context("Backup && Restore data on different Deployment", func() {
		BeforeEach(func() {
			pvc = f.GetPersistentVolumeClaim()
			err = f.CreatePersistentVolumeClaim(pvc)
			Expect(err).NotTo(HaveOccurred())
			deployment = f.Deployment(pvc.Name)

			pvc = f.GetPersistentVolumeClaim()
			err = f.CreatePersistentVolumeClaim(pvc)
			Expect(err).NotTo(HaveOccurred())
			recoveredDeployment = f.Deployment(pvc.Name)

			targetref = v1beta1.TargetRef{
				APIVersion: "apps/v1",
				Kind:       apis.KindDeployment,
				Name:       deployment.Name,
			}
		})
		AfterEach(func() {
			err := f.DeleteDeployment(deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err =framework.WaitUntilDeploymentDeleted(f.KubeClient,deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeleteDeployment(recoveredDeployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err =framework.WaitUntilDeploymentDeleted(f.KubeClient,recoveredDeployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})
		It("Restore data on different Deployment", func() {
            shouldBackupNewDeployment()
			By("Creating another Deployment " + recoveredDeployment.Name)
			_, err = f.CreateDeployment(recoveredDeployment)
			Expect(err).NotTo(HaveOccurred())
			err = util.WaitUntilDeploymentReady(f.KubeClient, recoveredDeployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			restoreSession.Spec.Target.Ref.Name = recoveredDeployment.Name
			By("Creating Restore Session")
			err = f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for initContainer")
			f.EventuallyDeployment(recoveredDeployment.ObjectMeta).Should(matcher.HaveInitContainer(util.StashInitContainer))

			By("Waiting for restore to succeed")
			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))

			By("checking the workload data has been restored")
			restoredData, err = f.ReadSampleDataFromMountedDirectory(recoveredDeployment.ObjectMeta, framework.GetPathsFromRestoreSession(&restoreSession), apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(restoredData).To(Not(BeEmpty()))

			By("Compare between restore data and sample data")
			Expect(restoredData).To(BeEquivalentTo(sampleData))

		})
	})
})
