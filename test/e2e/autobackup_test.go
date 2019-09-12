package e2e_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "kmodules.xyz/client-go/core/v1"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
)

var (
	backupBlueprint v1beta1.BackupBlueprint
)

var _ = Describe("Auto Backup", func() {
	BeforeEach(func() {
		f = root.Invoke()

		By("Ensure function")
		err = f.GetFunction()
		Expect(err).NotTo(HaveOccurred())
		By("Ensure Task")
		err = f.GetTask()
		Expect(err).NotTo(HaveOccurred())
	})
	JustBeforeEach(func() {
		cred = f.SecretForGCSBackend()
		if missing, _ := BeZero().Match(cred); missing {
			Skip("Missing repository credential")
		}
		backupBlueprint = f.GetBackupBlueprint(cred.Name)
	})
	AfterEach(func() {
		err = f.DeleteSecret(cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
		err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
	})
	var (
		testPVCAutoBackup = func() {
			By(fmt.Sprintf("Creating %s/%s storage Secret", cred.Namespace, cred.Name))
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Creating %s/%s BackupBlueprint", backupBlueprint.Namespace, backupBlueprint.Name))
			_, err = f.CreateBackupBlueprint(backupBlueprint)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Creating New %s/%s PVC", pvc.Namespace, pvc.Name))
			err = f.CreatePersistentVolumeClaim(pvc)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Creating %s/%s Pod", pod.Namespace, pod.Name))
			err = f.CreatePod(pod)
			Expect(err).NotTo(HaveOccurred())
			err = v1.WaitUntilPodRunning(f.KubeClient, pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Creating sample data inside Pod")
			err = f.CreateSampleDataInsideWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())

			By("Reading sample data from /source/data directory inside pod")
			sampleData, err = f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Adding annotations to the %s/%s PVC to run auto backup", pvc.Namespace, pvc.Name))
			err = f.AddAnnotationsToTarget(backupBlueprint.Name, pvc.ObjectMeta, apis.KindPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("validating updated %s/%s PVC has new annotation", pvc.Namespace, pvc.Name))
			f.EventuallyAnnotationsUpdated(backupBlueprint.Name, pvc.ObjectMeta, apis.KindPersistentVolumeClaim).Should(BeTrue())

			By("Waiting for created Repository")
			f.EventuallyRepositoryCreated(pvc.ObjectMeta).Should(BeTrue())
			repo, err = f.GetRepository(pvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for created BackupConfiguration")
			f.EventuallyBackupConfigurationCreated(pvc.ObjectMeta).Should(BeTrue())
			backupCfg, err = f.GetBackupConfiguration(pvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for BackupSession")
			f.EventuallyBackupSessionCreated(pvc.ObjectMeta).Should(BeTrue())
			backupSession, err := f.GetBackupSession(pvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Check for succeeded BackupSession")
			f.EventuallyBackupSessionPhase(backupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionSucceeded))

			By("Check for repository status updated")
			f.EventuallyRepository(&pvc).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
		}
		testDeploymentAutoBackup = func() {
			By(fmt.Sprintf("Creating %s/%s storage Secret", cred.Namespace, cred.Name))
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Creating %s/%s BackupBlueprint", backupBlueprint.Namespace, backupBlueprint.Name))
			_, err = f.CreateBackupBlueprint(backupBlueprint)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Creating New %s/%s Deployment", deployment.Namespace, deployment.Name))
			_, err = f.CreateDeployment(deployment)
			err = util.WaitUntilDeploymentReady(f.KubeClient, deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Creating sample data inside workload")
			err = f.CreateSampleDataInsideWorkload(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			By("Reading sample data from /source/data mountPath inside workload")
			sampleData, err = f.ReadSampleDataFromFromWorkload(deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleData).To(Not(BeEmpty()))

			By(fmt.Sprintf("Adding annotations to the %s/%s Deployment to run auto backup", deployment.Namespace, deployment.Name))
			err = f.AddAnnotationsToTarget(backupBlueprint.Name, deployment.ObjectMeta, apis.KindDeployment)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("validating updated %s/%s Deployment has new annotation", deployment.Namespace, deployment.Name))
			f.EventuallyAnnotationsUpdated(backupBlueprint.Name, deployment.ObjectMeta, apis.KindDeployment).Should(BeTrue())

			By("Waiting for created Repository")
			f.EventuallyRepositoryCreated(deployment.ObjectMeta).Should(BeTrue())
			repo, err = f.GetRepository(deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for created BackupConfiguration")
			f.EventuallyBackupConfigurationCreated(deployment.ObjectMeta).Should(BeTrue())
			backupCfg, err = f.GetBackupConfiguration(deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for BackupSession")
			f.EventuallyBackupSessionCreated(deployment.ObjectMeta).Should(BeTrue())
			backupSession, err := f.GetBackupSession(deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Check for succeeded BackupSession")
			f.EventuallyBackupSessionPhase(backupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionSucceeded))

			By("Check for repository status updated")
			f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
		}
		testSSAutoBackup = func() {
			By(fmt.Sprintf("Creating %s/%s storage Secret", cred.Namespace, cred.Name))
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Creating %s/%s BackupBlueprint", backupBlueprint.Namespace, backupBlueprint.Name))
			_, err = f.CreateBackupBlueprint(backupBlueprint)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Creating New %s/%s StatefulSet", ss.Namespace, ss.Name))
			_, err = f.CreateStatefulSet(ss)
			err = util.WaitUntilStatefulSetReady(f.KubeClient, ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Creating sample data inside workload")
			err = f.CreateSampleDataInsideWorkload(ss.ObjectMeta, apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())

			By("Reading sample data from /source/data mountPath inside workload")
			sampleData, err = f.ReadSampleDataFromFromWorkload(ss.ObjectMeta, apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleData).To(Not(BeEmpty()))

			By(fmt.Sprintf("Adding annotations to the %s/%s Stateful to run auto backup", ss.Namespace, ss.Name))
			err = f.AddAnnotationsToTarget(backupBlueprint.Name, ss.ObjectMeta, apis.KindStatefulSet)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("validating updated %s/%s Stateful has new annotation", ss.Namespace, ss.Name))
			f.EventuallyAnnotationsUpdated(backupBlueprint.Name, ss.ObjectMeta, apis.KindStatefulSet).Should(BeTrue())

			By("Waiting for created Repository")
			f.EventuallyRepositoryCreated(ss.ObjectMeta).Should(BeTrue())
			repo, err = f.GetRepository(ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for created BackupConfiguration")
			f.EventuallyBackupConfigurationCreated(ss.ObjectMeta).Should(BeTrue())
			backupCfg, err = f.GetBackupConfiguration(ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for BackupSession")
			f.EventuallyBackupSessionCreated(ss.ObjectMeta).Should(BeTrue())
			backupSession, err := f.GetBackupSession(ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Check for succeeded BackupSession")
			f.EventuallyBackupSessionPhase(backupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionSucceeded))

		}
	)
	Context("Auto Backup for PVC", func() {
		BeforeEach(func() {
			pvc = f.GetPersistentVolumeClaim()
			pod = f.Pod(pvc.Name)
		})
		JustBeforeEach(func() {
			backupBlueprint.Spec.Task = v1beta1.TaskRef{
				Name: framework.TaskPVCBackup,
			}
		})
		AfterEach(func() {
			err = f.DeleteBackupBlueprint(backupBlueprint.Name)
			err = framework.WaitUntilBackupBlueprintDeleted(f.StashClient, backupBlueprint.Name)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRepository(repo.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRepositoryDeleted(f.StashClient, backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteBackupConfiguration(backupCfg.ObjectMeta)
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err := f.DeletePersistentVolumeClaim(pvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		})
		It("General Auto Backup new PVC", func() {
			By("testing Auto Backup")
			testPVCAutoBackup()

		})
	})
	Context("Auto Backup for Deployment", func() {
		BeforeEach(func() {
			pvc = f.GetPersistentVolumeClaim()
			err = f.CreatePersistentVolumeClaim(pvc)
			Expect(err).NotTo(HaveOccurred())
			deployment = f.Deployment(pvc.Name)
		})
		AfterEach(func() {
			err = f.DeleteBackupBlueprint(backupBlueprint.Name)
			err = framework.WaitUntilBackupBlueprintDeleted(f.StashClient, backupBlueprint.Name)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRepository(repo.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRepositoryDeleted(f.StashClient, backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteBackupConfiguration(backupCfg.ObjectMeta)
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err := f.DeletePersistentVolumeClaim(pvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		})
		It("General Auto Backup new Deployment", func() {
			By("testing Auto Backup")
			testDeploymentAutoBackup()

		})
	})
	Context("Auto Backup for StatefulSet", func() {
		BeforeEach(func() {
			svc = f.HeadlessService()
			ss = f.StatefulSetForV1beta1API()
		})
		JustBeforeEach(func() {
			By("Creating service " + svc.Name)
			err = f.CreateOrPatchService(svc)
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			err = f.DeleteBackupBlueprint(backupBlueprint.Name)
			err = framework.WaitUntilBackupBlueprintDeleted(f.StashClient, backupBlueprint.Name)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRepository(ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRepositoryDeleted(f.StashClient, ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteBackupConfiguration(ss.ObjectMeta)
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err := f.DeletePersistentVolumeClaim(pvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		})
		FIt("General Auto Backup new StatefulSet", func() {
			By("testing Auto Backup")
			testSSAutoBackup()

		})
	})
})
