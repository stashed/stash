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
		err                 error
		f                   *framework.Invocation
		cred                core.Secret
		deployment          apps.Deployment
		recoveredDeployment apps.Deployment
		repo                *api.Repository
		backupCfg           v1beta1.BackupConfiguration
		restoreSession      v1beta1.RestoreSession
		pvc                 *core.PersistentVolumeClaim
		targetref           v1beta1.TargetRef
		rules          []v1beta1.Rule
	)
	var (
		sampleData   []string
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
		rules = []v1beta1.Rule{
			{
				Paths: []string{
					framework.TestSourceDataMountPath,
				},
			},
		}
		restoreSession = f.RestoreSession(repo.Name, targetref, rules)
	})
	AfterEach(func() {
		err = f.DeleteSecret(cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
		err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
	})
	var (
		testDeploymentBackup = func() {
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
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
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
			err = framework.WaitUntilDeploymentDeleted(f.KubeClient, deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		})
		It("General Backup new Deployment", func() {
			By("Creating New Deployment Backup")
			testDeploymentBackup()

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
			err = framework.WaitUntilDeploymentDeleted(f.KubeClient, deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
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
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
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
			err = framework.WaitUntilDeploymentDeleted(f.KubeClient, deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeleteDeployment(recoveredDeployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilDeploymentDeleted(f.KubeClient, recoveredDeployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})
		It("Restore data on different Deployment", func() {
			By("Creating New Deployment Backup")
			testDeploymentBackup()

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

var _ = Describe("Statefulset", func() {
	var (
		err            error
		f              *framework.Invocation
		cred           core.Secret
		ss             apps.StatefulSet
		recoveredss    apps.StatefulSet
		repo           *api.Repository
		backupCfg      v1beta1.BackupConfiguration
		restoreSession v1beta1.RestoreSession
		pvc            *core.PersistentVolumeClaim
		svc            core.Service
		targetref      v1beta1.TargetRef
		rules          []v1beta1.Rule
	)
	var (
		sampleData   []string
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
		restoreSession = f.RestoreSession(repo.Name, targetref, rules)
	})
	AfterEach(func() {
		err = f.DeleteSecret(cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
		err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
	})
	var (
		testStatefulsetBackup = func() {
			ss.Spec.Replicas = types.Int32P(3)
			By("Create Statefulset with multiple replica" + ss.Name)
			_, err := f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())
			err = util.WaitUntilStatefulSetReady(f.KubeClient, ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Sample data in inside pod")
			err = f.CreateSampleDataInsideWorkload(ss.ObjectMeta, apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())

			By("Reading sample data from /source/data mountPath inside workload")
			sampleData, err = f.ReadSampleDataFromFromWorkload(ss.ObjectMeta, apis.KindStatefulSet)
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
			f.EventuallyStatefulSet(ss.ObjectMeta).Should(matcher.HaveSidecar(util.StashContainer))

			By("Waiting for BackupSession")
			f.EventuallyBackupSessionCreated(backupCfg.ObjectMeta).Should(BeTrue())
			bs, err := f.GetBackupSession(backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Check for repository status updated")
			f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))

			By("Check for succeeded BackupSession")
			f.EventuallyBackupSessionPhase(bs.ObjectMeta).Should(Equal(v1beta1.BackupSessionSucceeded))

			By("Delete BackupConfiguration")
			err = f.DeleteBackupConfiguration(backupCfg)
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
			Expect(err).ShouldNot(HaveOccurred())

			By("Deleting sample data from pod")
			err = f.CleanupSampleDataFromWorkload(ss.ObjectMeta, apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())

		}
	)
	Context("General Backup new StatefulSet", func() {
		BeforeEach(func() {
			svc = f.HeadlessService()
			By("Creating service " + svc.Name)
			err = f.CreateService(svc)
			Expect(err).NotTo(HaveOccurred())

			ss = f.StatefulSetv1beta1()
			targetref = v1beta1.TargetRef{
				Name:       ss.Name,
				Kind:       apis.KindStatefulSet,
				APIVersion: "apps/v1",
			}
			rules = []v1beta1.Rule{
				{
					Paths: []string{
						framework.TestSourceDataMountPath,
					},
				},
			}
		})
		AfterEach(func() {
			err := f.DeleteStatefulSet(ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})

		It("General Backup new Statefulset", func() {
			By("Creating Statefulset Backup")
			testStatefulsetBackup()

			By("Creating Restore Session")
			err = f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())
			err = util.WaitUntilStatefulSetReady(f.KubeClient, ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for initContainer")
			f.EventuallyStatefulSet(ss.ObjectMeta).Should(matcher.HaveInitContainer(util.StashInitContainer))
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for restore to succeed")
			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))
			err = util.WaitUntilStatefulSetReady(f.KubeClient, ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("checking the workload data has been restored")
			restoredData, err = f.ReadSampleDataFromMountedDirectory(ss.ObjectMeta, framework.GetPathsFromRestoreSession(&restoreSession), apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())

			By("Compare between restore data and sample data")
			Expect(restoredData).To(BeEquivalentTo(sampleData))

		})
	})

	Context("Backup && Restore data on different StatefulSet", func() {
		BeforeEach(func() {
			svc = f.HeadlessService()
			ss = f.StatefulSetv1beta1()
			recoveredss = f.StatefulSetv1beta1()
			targetref = v1beta1.TargetRef{
				Name:       ss.Name,
				Kind:       apis.KindStatefulSet,
				APIVersion: "apps/v1",
			}
			rules = []v1beta1.Rule{
				{
					Paths: []string{
						framework.TestSourceDataMountPath,
					},
				},
			}
		})
		AfterEach(func() {
			err := f.DeleteStatefulSet(ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeleteStatefulSet(recoveredss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, recoveredss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})

		It("General Backup new Statefulset", func() {
			By("Creating Statefulset Backup")
			testStatefulsetBackup()

			By("Creating another StatefulSet " + recoveredss.Name)
			_, err := f.CreateStatefulSet(recoveredss)
			Expect(err).NotTo(HaveOccurred())

			restoreSession.Spec.Target.Ref.Name = recoveredss.Name

			By("Creating Restore Session")
			err = f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())
			err = util.WaitUntilStatefulSetReady(f.KubeClient, recoveredss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for initContainer")
			f.EventuallyStatefulSet(recoveredss.ObjectMeta).Should(matcher.HaveInitContainer(util.StashInitContainer))
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for restore to succeed")
			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))
			err = util.WaitUntilStatefulSetReady(f.KubeClient, recoveredss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("checking the workload data has been restored")
			restoredData, err = f.ReadSampleDataFromMountedDirectory(recoveredss.ObjectMeta, framework.GetPathsFromRestoreSession(&restoreSession), apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())

			By("Compare between restore data and sample data")
			Expect(restoredData).To(BeEquivalentTo(sampleData))

		})
	})

	Context("Restore data on Scaled Up StatefulSet", func() {
		BeforeEach(func() {
			svc = f.HeadlessService()
			ss = f.StatefulSetv1beta1()
			recoveredss = f.StatefulSetv1beta1()
			targetref = v1beta1.TargetRef{
				Name:       ss.Name,
				Kind:       apis.KindStatefulSet,
				APIVersion: "apps/v1",
			}
			rules = []v1beta1.Rule{
				{
					Subjects: []string{
						"host-3",
						"host-4",
					},
					SourceHost: "host-1",
					Paths: []string{
						framework.TestSourceDataMountPath,
					},
				},
				{
					Subjects:   []string{},
					SourceHost: "",
					Paths: []string{
						framework.TestSourceDataMountPath,
					},
				},
			}
		})
		AfterEach(func() {
			err := f.DeleteStatefulSet(ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeleteStatefulSet(recoveredss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, recoveredss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})

		It("General Backup new Statefulset", func() {
			By("Creating Statefulset Backup")
			testStatefulsetBackup()

			By("Creating another StatefulSet " + recoveredss.Name)
			recoveredss.Spec.Replicas = types.Int32P(5)
			_, err := f.CreateStatefulSet(recoveredss)
			Expect(err).NotTo(HaveOccurred())

			restoreSession.Spec.Target.Ref.Name = recoveredss.Name

			By("Creating Restore Session")
			err = f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())
			err = util.WaitUntilStatefulSetReady(f.KubeClient, recoveredss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for initContainer")
			f.EventuallyStatefulSet(recoveredss.ObjectMeta).Should(matcher.HaveInitContainer(util.StashInitContainer))
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for restore to succeed")
			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))
			err = util.WaitUntilStatefulSetReady(f.KubeClient, recoveredss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("checking the workload data has been restored")
			restoredData, err = f.ReadSampleDataFromMountedDirectory(recoveredss.ObjectMeta, framework.GetPathsFromRestoreSession(&restoreSession), apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())

			By("Comparing between first and second StatefulSet sample data")
			Expect(sampleData).Should(BeEquivalentTo(restoredData[0:(len(restoredData)-len(sampleData))+1]))
			data := make([]string, 0)
			data = append(data, sampleData[1])
			data = append(data, sampleData[1])
			Expect(data).Should(BeEquivalentTo(restoredData[(len(restoredData)-len(sampleData)+1):]))

		})
	})

})
