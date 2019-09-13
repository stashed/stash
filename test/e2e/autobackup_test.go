package e2e_test

import (
	"fmt"

	"stash.appscode.dev/stash/apis/stash/v1alpha1"

	v1 "kmodules.xyz/objectstore-api/api/v1"

	"stash.appscode.dev/stash/test/e2e/matcher"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	apps_util "kmodules.xyz/client-go/apps/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
)

var (
	backupBlueprint v1beta1.BackupBlueprint
)

var _ = Describe("PVC", func() {
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
		backupBlueprint = f.BackupBlueprint(cred.Name)
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
			err = core_util.WaitUntilPodRunning(f.KubeClient, pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Creating sample data inside Pod")
			err = f.CreateSampleDataInsideWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())

			By("Reading sample data from /source/data directory inside pod")
			sampleData, err = f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleData).To(Not(BeEmpty()))

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
	)
	Context("Should success AutoBackup for PVC", func() {
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
			err = f.DeletePod(pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilPodDeleted(f.KubeClient, pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err := f.DeletePersistentVolumeClaim(pvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteBackupBlueprint(backupBlueprint.Name)
			err = framework.WaitUntilBackupBlueprintDeleted(f.StashClient, backupBlueprint.Name)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteBackupConfiguration(backupCfg.ObjectMeta)
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRepository(repo.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRepositoryDeleted(f.StashClient, repo.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})
		It("Run AutoBackup for new PVC", func() {
			By("Checking AutoBackup is succeeded")
			testPVCAutoBackup()

		})
	})
	Context("Should fail AutoBackup for Missing BackupBlueprint credential", func() {

	})
	Context("Should fail AutoBackup for Missing Annotation in PVC", func() {
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
		})
		It("Should fail AutoBackup for missing/giving wrong BackupBlueprint name as annotations in PVC", func() {
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
			err = core_util.WaitUntilPodRunning(f.KubeClient, pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Creating sample data inside Pod")
			err = f.CreateSampleDataInsideWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())

			By("Reading sample data from /source/data directory inside pod")
			sampleData, err = f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())
			Expect(sampleData).To(Not(BeEmpty()))

			By(fmt.Sprintf("Adding Wrong(BackupBlueprint name) annotations to the %s/%s PVC to run AutoBackup", pvc.Namespace, pvc.Name))
			wrongBackupBlueprintName := "backup-blueprint"
			pvc, _, err = core_util.PatchPVC(f.KubeClient, pvc, func(in *core.PersistentVolumeClaim) *core.PersistentVolumeClaim {
				in.SetAnnotations(map[string]string{
					v1beta1.KeyBackupBlueprint: wrongBackupBlueprintName,
				})
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			By("Should not get respective BackupBlueprint because of adding wrong annotations in PVC")
			annotations := pvc.GetAnnotations()
			fmt.Println(annotations)
			_, err = f.GetBackupBlueprint(annotations[v1beta1.KeyBackupBlueprint])
			Expect(err).To(HaveOccurred())

			err = f.DeletePod(pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilPodDeleted(f.KubeClient, pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err := f.DeletePersistentVolumeClaim(pvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

var _ = Describe("Deployment", func() {
	BeforeEach(func() {
		f = root.Invoke()
	})
	JustBeforeEach(func() {
		cred = f.SecretForGCSBackend()
		if missing, _ := BeZero().Match(cred); missing {
			Skip("Missing repository credential")
		}
		backupBlueprint = f.BackupBlueprint(cred.Name)
	})
	AfterEach(func() {
		err = f.DeleteSecret(cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
		err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
	})
	var (
		AutoBackupSucceededTest = func() {
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
	)
	Context("Should success AutoBackup for Deployment", func() {
		BeforeEach(func() {
			pvc = f.GetPersistentVolumeClaim()
			err = f.CreatePersistentVolumeClaim(pvc)
			Expect(err).NotTo(HaveOccurred())
			deployment = f.Deployment(pvc.Name)
		})
		AfterEach(func() {
			err = f.DeleteDeployment(deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilDeploymentDeleted(f.KubeClient, deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteBackupBlueprint(backupBlueprint.Name)
			err = framework.WaitUntilBackupBlueprintDeleted(f.StashClient, backupBlueprint.Name)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteBackupConfiguration(backupCfg.ObjectMeta)
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRepository(repo.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRepositoryDeleted(f.StashClient, repo.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err := f.DeletePersistentVolumeClaim(pvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		})
		It("Run AutoBackup for new Deployment", func() {
			By("Checking AutoBackup is succeeded")
			AutoBackupSucceededTest()

		})
	})
	Context("Should fail AutoBackup for Missing BackupBlueprint credential", func() {
		BeforeEach(func() {
			pvc = f.GetPersistentVolumeClaim()
			err = f.CreatePersistentVolumeClaim(pvc)
			Expect(err).NotTo(HaveOccurred())
			deployment = f.Deployment(pvc.Name)
		})
		JustBeforeEach(func() {
			backupBlueprint = f.BackupBlueprint(cred.Name)
		})
		AfterEach(func() {
			err = f.DeleteBackupBlueprint(backupBlueprint.Name)
			err = framework.WaitUntilBackupBlueprintDeleted(f.StashClient, backupBlueprint.Name)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteBackupConfiguration(backupCfg.ObjectMeta)
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRepository(repo.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRepositoryDeleted(f.StashClient, repo.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err := f.DeletePersistentVolumeClaim(pvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		})
		It("Should fail AutoBackup for missing Repository secret in BackupBlueprint", func() {
			By(fmt.Sprintf("Creating %s/%s storage Secret", cred.Namespace, cred.Name))
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Adding empty/wrong Repository Secret in BackupBlueprint")
			backupBlueprint.Spec.Backend.StorageSecretName = ""

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
			_, err := f.GetBackupSession(deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Should fail Sidecar injection because of missing Repository Secret")
			f.EventuallyDeployment(deployment.ObjectMeta).ShouldNot(matcher.HaveSidecar(util.StashContainer))

		})
		It("Should fail AutoBackup for missing Repository Backend in BackupBlueprint", func() {
			By(fmt.Sprintf("Creating %s/%s storage Secret", cred.Namespace, cred.Name))
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Adding empty/wrong Repository Backend in BackupBlueprint")
			gcsbackend := v1.GCSSpec{}
			backupBlueprint.Spec.Backend.GCS = &gcsbackend

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

			By("Should fail BackupSession for wrong/missing backend credential in BackupBlueprint")
			f.EventuallyBackupSessionPhase(backupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionFailed))

		})
		It("Should fail AutoBackup for missing BackupConfiguration RetentionPolicy in BackupBlueprint", func() {
			By(fmt.Sprintf("Creating %s/%s storage Secret", cred.Namespace, cred.Name))
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Adding empty/wrong RetentionPolicy in BackupBlueprint")
			backupBlueprint.Spec.RetentionPolicy = v1alpha1.RetentionPolicy{}

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

			By("Should fail BackupSession for missing RetentionPolicy in BackupBlueprint")
			f.EventuallyBackupSessionPhase(backupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionFailed))
		})
		It("Should fail AutoBackup for missing BackupConfiguration Schedule in BackupBlueprint", func() {
			By(fmt.Sprintf("Creating %s/%s storage Secret", cred.Namespace, cred.Name))
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Adding empty/wrong RetentionPolicy in BackupBlueprint")
			backupBlueprint.Spec.Schedule = ""

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

			By("Should fail creating BackupSession for missing Schedule in BackupBlueprint")
			f.EventuallyBackupSessionNotCreated(deployment.ObjectMeta).Should(BeTrue())
			_, err := f.GetBackupSession(deployment.ObjectMeta)
			Expect(err).To(HaveOccurred())
		})

	})
	Context("Should failed AutoBackup for Missing Annotation in Deployment", func() {
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

			err := f.DeletePersistentVolumeClaim(pvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})
		It("Should fail AutoBackup for missing/giving wrong BackupBlueprint name as annotations in Deployment", func() {
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

			By(fmt.Sprintf("Adding Wrong(BackupBlueprint name) annotations to the %s/%s Deployment to run AutoBackup", deployment.Namespace, deployment.Name))
			wrongBackupBlueprintName := "backup-blueprint"
			deployment, _, err := apps_util.PatchDeployment(f.KubeClient, &deployment, func(in *apps.Deployment) *apps.Deployment {
				in.SetAnnotations(map[string]string{
					v1beta1.KeyBackupBlueprint: wrongBackupBlueprintName,
					v1beta1.KeyTargetPaths:     "/source/data",
					v1beta1.KeyVolumeMounts:    "source-data:/source/data",
				})
				return in
			})
			Expect(err).NotTo(HaveOccurred())

			By(" Should not get respective BackupBlueprint because of adding wrong annotations in Deployment")
			annotations := deployment.Annotations
			fmt.Println(annotations)
			_, err = f.GetBackupBlueprint(annotations[v1beta1.KeyBackupBlueprint])
			Expect(err).To(HaveOccurred())

		})
		It("Should fail AutoBackup for missing/giving wrong TargetPath in Deployment annotations", func() {
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

			By(fmt.Sprintf("Adding Wrong(TargetPath) annotations to the %s/%s Deployment to run AutoBackup", deployment.Namespace, deployment.Name))
			_, _, err = apps_util.PatchDeployment(f.KubeClient, &deployment, func(in *apps.Deployment) *apps.Deployment {
				wrongTargetPath := "/source/data-1"
				in.SetAnnotations(map[string]string{
					v1beta1.KeyBackupBlueprint: backupBlueprint.Name,
					v1beta1.KeyTargetPaths:     wrongTargetPath,
					v1beta1.KeyVolumeMounts:    "source-data:/source/data",
				})
				return in
			})
			Expect(err).NotTo(HaveOccurred())

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

			By("Should fail BackupSession for adding wrong TargetPath as annotations in Deployment")
			f.EventuallyBackupSessionPhase(backupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionFailed))

			err = f.DeleteDeployment(deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilDeploymentDeleted(f.KubeClient, backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteBackupConfiguration(backupCfg.ObjectMeta)
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRepository(repo.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRepositoryDeleted(f.StashClient, repo.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		})
		It("Should fail AutoBackup for missing/giving wrong MountPath in Deployment annotations", func() {
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

			By(fmt.Sprintf("Adding Wrong(TargetPath) annotations to the %s/%s Deployment to run AutoBackup", deployment.Namespace, deployment.Name))
			_, _, err = apps_util.PatchDeployment(f.KubeClient, &deployment, func(in *apps.Deployment) *apps.Deployment {
				wrongMountPath := "source-data:/source/data-1"
				in.SetAnnotations(map[string]string{
					v1beta1.KeyBackupBlueprint: backupBlueprint.Name,
					v1beta1.KeyTargetPaths:     "/source/data",
					v1beta1.KeyVolumeMounts:    wrongMountPath,
				})
				return in
			})
			Expect(err).NotTo(HaveOccurred())

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

			By("Should fail BackupSession for adding wrong MountPath as annotations in Deployment")
			f.EventuallyBackupSessionPhase(backupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionFailed))

			err = f.DeleteDeployment(deployment.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilDeploymentDeleted(f.KubeClient, backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteBackupConfiguration(backupCfg.ObjectMeta)
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRepository(repo.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRepositoryDeleted(f.StashClient, repo.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		})
	})
})

var _ = Describe("StatefulSet", func() {
	BeforeEach(func() {
		f = root.Invoke()
	})
	JustBeforeEach(func() {
		cred = f.SecretForGCSBackend()
		if missing, _ := BeZero().Match(cred); missing {
			Skip("Missing repository credential")
		}
		backupBlueprint = f.BackupBlueprint(cred.Name)
	})
	AfterEach(func() {
		err = f.DeleteSecret(cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
		err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
	})
	var (
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
			err = f.DeleteStatefulSet(ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteBackupBlueprint(backupBlueprint.Name)
			err = framework.WaitUntilBackupBlueprintDeleted(f.StashClient, backupBlueprint.Name)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteBackupConfiguration(ss.ObjectMeta)
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, ss.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRepository(repo.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRepositoryDeleted(f.StashClient, repo.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		})
		It("General Auto Backup new StatefulSet", func() {
			By("testing Auto Backup")
			testSSAutoBackup()

		})
	})

})
