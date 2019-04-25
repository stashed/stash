package e2e_test

import (
	"fmt"

	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/appscode/stash/pkg/util"
	"github.com/appscode/stash/test/e2e/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
)

var (
	bpv              *core.PersistentVolume
	bpvc             *core.PersistentVolumeClaim
	rpvc             *core.PersistentVolumeClaim
	pod              core.Pod
	updateStatusFunc v1beta1.Function
	backupFunc       v1beta1.Function
	restoreFunc      v1beta1.Function
	backupTask       v1beta1.Task
	restoreTask      v1beta1.Task
)

var _ = Describe("Volume", func() {
	BeforeEach(func() {
		f = root.Invoke()

	})
	JustBeforeEach(func() {
		pod = f.Pod(bpvc.Name)
		cred = f.SecretForLocalBackend()
		if missing, _ := BeZero().Match(cred); missing {
			Skip("Missing repository credential")
		}
		pvc = f.GetPersistentVolumeClaim()
		err = f.CreatePersistentVolumeClaim(pvc)
		Expect(err).NotTo(HaveOccurred())
		repo = f.Repository(cred.Name, pvc.Name)
		fmt.Println("pvc for repo", pvc.Name)
		targetref = v1beta1.TargetRef{}
		backupCfg = f.BackupConfiguration(repo.Name, targetref)
		backupCfg.Spec.Target = &v1beta1.Target{
			Ref: v1beta1.TargetRef{
				APIVersion: "v1",
				Kind:       apis.KindPersistentVolumeClaim,
				Name:       bpvc.Name,
			},
			VolumeMounts: []core.VolumeMount{
				{
					Name:      framework.TestSourceDataVolumeName,
					MountPath: framework.TestSourceDataMountPath,
				},
			},
			Directories: []string{
				framework.TestSourceDataMountPath,
			},
		}
		backupCfg.Spec.Task.Name = backupTask.Name

		restoreSession = f.RestoreSession(repo.Name, targetref, rules)
		restoreSession.Spec.Target = &v1beta1.Target{
			Ref: v1beta1.TargetRef{
				APIVersion: "v1",
				Kind:       apis.KindPersistentVolumeClaim,
				Name:       bpvc.Name,
			},
			VolumeMounts: []core.VolumeMount{
				{
					Name:      framework.TestSourceDataVolumeName,
					MountPath: framework.TestSourceDataMountPath,
				},
			},
		}
		restoreSession.Spec.Rules = []v1beta1.Rule{
			{
				Paths: []string{
					framework.TestSourceDataMountPath,
				},
			},
		}
		restoreSession.Spec.Task.Name = restoreTask.Name

	})
	AfterEach(func() {
		err = f.DeleteSecret(cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
		err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
	})
	var (
		testPVCBackup = func() {
			By("Creating New PV and PVC")
			err = f.CreatePersistentVolume(bpv)
			Expect(err).NotTo(HaveOccurred())

			err = f.CreatePersistentVolumeClaim(bpvc)
			Expect(err).NotTo(HaveOccurred())

			By("Create Pod and Generate sample Data")
			err = f.CreatePod(pod)
			Expect(err).NotTo(HaveOccurred())
			err = util.WaitUntilPodRunning(f.KubeClient, pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Reading sample data from /source/data directory inside pod")
			sampleData, err = f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
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

			By("Waiting for BackupSession")
			f.EventuallyBackupSessionCreated(backupCfg.ObjectMeta).Should(BeTrue())
			bs, err := f.GetBackupSession(backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Check for succeeded BackupSession")
			f.EventuallyBackupSessionPhase(bs.ObjectMeta).Should(Equal(v1beta1.BackupSessionSucceeded))

			By("Delete BackupConfiguration")
			err = f.DeleteBackupConfiguration(backupCfg)
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Remove sample data from PVC")
			err = f.CleanupSampleDataFromWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())
			err = util.WaitUntilPodRunning(f.KubeClient, pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		}
	)
	Context("General Backup && Restore for PVC Volume", func() {
		BeforeEach(func() {
			bpv = f.GetPersistentVolume()
			bpvc = f.GetPersistentVolumeClaim()
			fmt.Println(bpvc.Name)

			updateStatusFunc = f.UpdateStatusFunction()
			backupFunc = f.BackupFunction()
			restoreFunc = f.RestoreFunction()

			err = f.CreateFunction(updateStatusFunc)
			Expect(err).NotTo(HaveOccurred())
			err = f.CreateFunction(backupFunc)
			Expect(err).NotTo(HaveOccurred())
			err = f.CreateFunction(restoreFunc)
			Expect(err).NotTo(HaveOccurred())

			backupTask = f.BackupTask()
			restoreTask = f.RestoreTask()

			err = f.CreateTask(backupTask)
			Expect(err).NotTo(HaveOccurred())
			err = f.CreateTask(restoreTask)
			Expect(err).NotTo(HaveOccurred())

		})
		AfterEach(func() {
			err = f.DeleteFunction(updateStatusFunc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeleteFunction(backupFunc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeleteFunction(restoreFunc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteTask(backupTask.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeleteTask(restoreTask.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeletePersistentVolume(bpv.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeletePersistentVolumeClaim(bpvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRepository(repo)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRepositoryDeleted(f.StashClient, repo)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRestoreSessionDeleted(f.StashClient, restoreSession.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		})
		It("General Backup new PVC", func() {
			By("new backup for PVC")
			testPVCBackup()

			By("Creating Restore Session")
			err = f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for restore to succeed")
			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))

			By("Reading sample data from /source/data directory inside pod")
			restoredData, err = f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeletePod(pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying restored data is same as original data")
			Expect(sampleData).To(BeEquivalentTo(restoredData))

		})
	})

	Context("Restore data on different PVC", func() {
		BeforeEach(func() {
			bpv = f.GetPersistentVolume()
			bpvc = f.GetPersistentVolumeClaim()
			rpvc = f.GetPersistentVolumeClaim()

			updateStatusFunc = f.UpdateStatusFunction()
			backupFunc = f.BackupFunction()
			restoreFunc = f.RestoreFunction()

			err = f.CreateFunction(updateStatusFunc)
			Expect(err).NotTo(HaveOccurred())
			err = f.CreateFunction(backupFunc)
			Expect(err).NotTo(HaveOccurred())
			err = f.CreateFunction(restoreFunc)
			Expect(err).NotTo(HaveOccurred())

			backupTask = f.BackupTask()
			restoreTask = f.RestoreTask()

			err = f.CreateTask(backupTask)
			Expect(err).NotTo(HaveOccurred())
			err = f.CreateTask(restoreTask)
			Expect(err).NotTo(HaveOccurred())

		})
		AfterEach(func() {
			err = f.DeleteFunction(updateStatusFunc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeleteFunction(backupFunc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeleteFunction(restoreFunc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteTask(backupTask.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeleteTask(restoreTask.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeletePersistentVolume(bpv.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeletePersistentVolumeClaim(bpvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeletePersistentVolumeClaim(rpvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRepository(repo)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRepositoryDeleted(f.StashClient, repo)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRestoreSessionDeleted(f.StashClient, restoreSession.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		})
		FIt("General Backup new PVC", func() {
			By("new backup for PVC")
			testPVCBackup()

			By("Create another PVC")
			err := f.CreatePersistentVolumeClaim(rpvc)

			By("Creating Restore Session")
			restoreSession.Spec.Target.Ref.Name = rpvc.Name
			err = f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for restore to succeed")
			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))

			By("delete previous Pod")
			err = f.DeletePod(pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			By("Create Pod and Generate sample Data")
			pod.Name = rand.WithUniqSuffix("restore-test")
			pod.Spec.Containers[0].Args[0] = "set -x; while true; do sleep 30; done;"
			pod.Spec.Volumes[0].PersistentVolumeClaim.ClaimName = rpvc.Name
			err = f.CreatePod(pod)
			Expect(err).NotTo(HaveOccurred())
			err = util.WaitUntilPodRunning(f.KubeClient, pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Reading sample data from /source/data directory inside pod")
			restoredData, err = f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeletePod(pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying restored data is same as original data")
			Expect(sampleData).To(BeEquivalentTo(restoredData))

		})
	})
})
