package e2e_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	apps_util "kmodules.xyz/client-go/apps/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	v1 "kmodules.xyz/objectstore-api/api/v1"
	"stash.appscode.dev/stash/apis"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
	"stash.appscode.dev/stash/pkg/util"
	"stash.appscode.dev/stash/test/e2e/framework"
)

var _ = Describe("Auto-Backup", func() {
	var (
		cred core.Secret
		f    *framework.Invocation
	)

	var _ = Describe("Deployment", func() {
		var (
			pvc        *core.PersistentVolumeClaim
			deployment apps.Deployment

			testDeploymentAutoBackupSucceeded = func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s ", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating BackupBlueprint: %s ", f.BackupBlueprint.Name))
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating Deployment: %s/%s", deployment.Namespace, deployment.Name))
				_, err = f.CreateDeployment(deployment)
				err = util.WaitUntilDeploymentReady(f.KubeClient, deployment.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Create and Ensure sample data inside running workload")
				err = f.CreateAndEnsureSampleDataInsideWorkload(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Adding auto-backup specific annotations to the Deployment: %s/%s", deployment.Namespace, deployment.Name))
				err = f.AddAutoBackupAnnotationsToTarget(f.BackupBlueprint.Name, deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the auto-backup annotations has been added successfully")
				f.EventuallyAutoBackupAnnotationsFound(f.BackupBlueprint.Name, deployment.ObjectMeta, apis.KindDeployment).Should(BeTrue())

				By("Waiting for Repository")
				f.EventuallyRepositoryCreated(f.Namespace()).Should(BeTrue())
				f.Repository, err = f.GetRepository(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupConfiguration")
				f.EventuallyBackupConfigurationCreated(f.Namespace()).Should(BeTrue())
				f.BackupConfig, err = f.GetBackupConfiguration(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession")
				f.EventuallyBackupSessionCreated(f.Namespace()).Should(BeTrue())
				f.BackupSession, err = f.GetBackupSession(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession to be succeeded")
				f.EventuallyBackupSessionPhase(f.BackupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionSucceeded))

				By("Check for repository status to be updated")
				f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
			}
		)
		BeforeEach(func() {
			f = root.Invoke()
		})
		JustBeforeEach(func() {
			cred = f.SecretForGCSBackend()
			if missing, _ := BeZero().Match(cred); missing {
				Skip("Missing repository credential")
			}
			f.BackupBlueprint = f.BackupBlueprintObj(cred.Name)
		})
		AfterEach(func() {
			err := f.DeleteSecret(cred.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})
		Context("Success event: ", func() {
			BeforeEach(func() {
				pvc = f.GetPersistentVolumeClaim()
				err := f.CreatePersistentVolumeClaim(pvc)
				Expect(err).NotTo(HaveOccurred())
				deployment = f.Deployment(pvc.Name)
			})
			AfterEach(func() {
				err := f.DeleteDeployment(deployment.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilDeploymentDeleted(f.KubeClient, deployment.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupBlueprint(f.BackupBlueprint.Name)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilBackupBlueprintDeleted(f.StashClient, f.BackupBlueprint.Name)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupConfiguration(f.BackupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, f.BackupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteRepository(f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilRepositoryDeleted(f.StashClient, f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeletePersistentVolumeClaim(pvc.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

			})
			It("Should success auto-backup for the Deployment", func() {
				testDeploymentAutoBackupSucceeded()
			})
		})
		Context("Failure event: ", func() {
			BeforeEach(func() {
				pvc = f.GetPersistentVolumeClaim()
				err := f.CreatePersistentVolumeClaim(pvc)
				Expect(err).NotTo(HaveOccurred())
				deployment = f.Deployment(pvc.Name)
			})
			AfterEach(func() {
				err := f.DeleteDeployment(deployment.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilDeploymentDeleted(f.KubeClient, deployment.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupBlueprint(f.BackupBlueprint.Name)
				err = framework.WaitUntilBackupBlueprintDeleted(f.StashClient, f.BackupBlueprint.Name)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupConfiguration(f.BackupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, f.BackupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteRepository(f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilRepositoryDeleted(f.StashClient, f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeletePersistentVolumeClaim(pvc.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

			})
			It("Should fail auto-backup for adding inappropriate Repository secret in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.Backend.StorageSecretName = ""
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).To(HaveOccurred())
			})
			It("Should fail auto-backup for adding inappropriate Repository backend in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.Backend.GCS = &v1.GCSSpec{}
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating Deployment: %s/%s", deployment.Namespace, deployment.Name))
				_, err = f.CreateDeployment(deployment)
				err = util.WaitUntilDeploymentReady(f.KubeClient, deployment.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Create and Ensure sample data inside running workload")
				err = f.CreateAndEnsureSampleDataInsideWorkload(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Adding auto-backup specific annotations to the Deployment: %s/%s", deployment.Namespace, deployment.Name))
				err = f.AddAutoBackupAnnotationsToTarget(f.BackupBlueprint.Name, deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the auto-backup annotations has been added successfully")
				f.EventuallyAutoBackupAnnotationsFound(f.BackupBlueprint.Name, deployment.ObjectMeta, apis.KindDeployment).Should(BeTrue())

				By("Waiting for Repository")
				f.EventuallyRepositoryCreated(f.Namespace()).Should(BeTrue())
				f.Repository, err = f.GetRepository(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupConfiguration")
				f.EventuallyBackupConfigurationCreated(f.Namespace()).Should(BeTrue())
				f.BackupConfig, err = f.GetBackupConfiguration(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession")
				f.EventuallyBackupSessionCreated(f.Namespace()).Should(BeTrue())
				f.BackupSession, err = f.GetBackupSession(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("waiting for BackupSession to be failed")
				f.EventuallyBackupSessionPhase(f.BackupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionFailed))

			})
			It("Should fail auto-backup for adding inappropriate BackupConfiguration RetentionPolicy in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.RetentionPolicy = v1alpha1.RetentionPolicy{}
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).To(HaveOccurred())
			})
			It("Should fail auto-backup for adding inappropriate BackupConfiguration Schedule in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.Schedule = ""
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).To(HaveOccurred())
			})

			It("Should fail auto-backup for adding inappropriate BackupBlueprint annotation in Deployment", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s ", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating BackupBlueprint: %s ", f.BackupBlueprint.Name))
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating Deployment: %s/%s", deployment.Namespace, deployment.Name))
				_, err = f.CreateDeployment(deployment)
				err = util.WaitUntilDeploymentReady(f.KubeClient, deployment.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Create and Ensure sample data inside running workload")
				err = f.CreateAndEnsureSampleDataInsideWorkload(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Adding auto-backup inappropriate annotations to the Deployment: %s/%s", deployment.Namespace, deployment.Name))
				wrongBackupBlueprintName := "backup-blueprint"
				deployment, _, err := apps_util.PatchDeployment(f.KubeClient, &deployment, func(in *apps.Deployment) *apps.Deployment {
					in.SetAnnotations(map[string]string{
						v1beta1.KeyBackupBlueprint: wrongBackupBlueprintName,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					})
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the auto-backup annotations has been added successfully")
				Expect(deployment.Annotations[v1beta1.KeyBackupBlueprint]).To(Equal(wrongBackupBlueprintName))
				Expect(deployment.Annotations[v1beta1.KeyTargetPaths]).To(Equal(framework.TestSourceDataTargetPath))
				Expect(deployment.Annotations[v1beta1.KeyVolumeMounts]).To(Equal(framework.TestSourceDataVolumeMount))

				By("Will fail to get respective BackupBlueprint")
				annotations := deployment.Annotations
				_, err = f.GetBackupBlueprint(annotations[v1beta1.KeyBackupBlueprint])
				Expect(err).To(HaveOccurred())

			})
			It("Should fail auto-backup for adding inappropriate TargetPath/MountPath annotations in Deployment", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s ", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating BackupBlueprint: %s ", f.BackupBlueprint.Name))
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating Deployment: %s/%s", deployment.Namespace, deployment.Name))
				_, err = f.CreateDeployment(deployment)
				err = util.WaitUntilDeploymentReady(f.KubeClient, deployment.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Create and Ensure sample data inside running workload")
				err = f.CreateAndEnsureSampleDataInsideWorkload(deployment.ObjectMeta, apis.KindDeployment)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Adding auto-backup inappropriate annotations to the Deployment: %s/%s", deployment.Namespace, deployment.Name))
				wrongTargetPath := "/source/data-1"
				deployment, _, err := apps_util.PatchDeployment(f.KubeClient, &deployment, func(in *apps.Deployment) *apps.Deployment {
					in.SetAnnotations(map[string]string{
						v1beta1.KeyBackupBlueprint: f.BackupBlueprint.Name,
						v1beta1.KeyTargetPaths:     wrongTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					})
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the auto-backup annotations has been added successfully")
				Expect(deployment.Annotations[v1beta1.KeyBackupBlueprint]).To(Equal(f.BackupBlueprint.Name))
				Expect(deployment.Annotations[v1beta1.KeyTargetPaths]).To(Equal(wrongTargetPath))
				Expect(deployment.Annotations[v1beta1.KeyVolumeMounts]).To(Equal(framework.TestSourceDataVolumeMount))

				By("Waiting for Repository")
				f.EventuallyRepositoryCreated(f.Namespace()).Should(BeTrue())
				f.Repository, err = f.GetRepository(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupConfiguration")
				f.EventuallyBackupConfigurationCreated(f.Namespace()).Should(BeTrue())
				f.BackupConfig, err = f.GetBackupConfiguration(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession")
				f.EventuallyBackupSessionCreated(f.Namespace()).Should(BeTrue())
				f.BackupSession, err = f.GetBackupSession(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession to be failed")
				f.EventuallyBackupSessionPhase(f.BackupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionFailed))
			})

		})
	})

	var _ = Describe("StatefulSet", func() {
		var (
			statefulSet apps.StatefulSet
			service     core.Service

			testStatefulSetAutoBackupSucceeded = func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating BackupBlueprint: %s", f.BackupBlueprint.Name))
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating StatefulSet: %s/%s", statefulSet.Namespace, statefulSet.Name))
				_, err = f.CreateStatefulSet(statefulSet)
				err = util.WaitUntilStatefulSetReady(f.KubeClient, statefulSet.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Create and Ensure sample data inside running workload")
				err = f.CreateAndEnsureSampleDataInsideWorkload(statefulSet.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Adding auto-backup specific annotations to the StatefulSet: %s/%s", statefulSet.Namespace, statefulSet.Name))
				err = f.AddAutoBackupAnnotationsToTarget(f.BackupBlueprint.Name, statefulSet.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the auto-backup annotations has been added successfully")
				f.EventuallyAutoBackupAnnotationsFound(f.BackupBlueprint.Name, statefulSet.ObjectMeta, apis.KindStatefulSet).Should(BeTrue())

				By("Waiting for Repository")
				f.EventuallyRepositoryCreated(f.Namespace()).Should(BeTrue())
				f.Repository, err = f.GetRepository(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupConfiguration")
				f.EventuallyBackupConfigurationCreated(f.Namespace()).Should(BeTrue())
				f.BackupConfig, err = f.GetBackupConfiguration(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession")
				f.EventuallyBackupSessionCreated(f.Namespace()).Should(BeTrue())
				backupSession, err := f.GetBackupSession(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession to be succeeded")
				f.EventuallyBackupSessionPhase(backupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionSucceeded))

			}
		)
		BeforeEach(func() {
			f = root.Invoke()
		})
		JustBeforeEach(func() {
			cred = f.SecretForGCSBackend()
			if missing, _ := BeZero().Match(cred); missing {
				Skip("Missing repository credential")
			}
			f.BackupBlueprint = f.BackupBlueprintObj(cred.Name)
		})
		AfterEach(func() {
			err := f.DeleteSecret(cred.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("Success event: ", func() {
			BeforeEach(func() {
				service = f.HeadlessService()
				statefulSet = f.StatefulSetForV1beta1API()
			})
			JustBeforeEach(func() {
				By("Creating service " + service.Name)
				err := f.CreateOrPatchService(service)
				Expect(err).NotTo(HaveOccurred())
			})
			AfterEach(func() {
				err := f.DeleteStatefulSet(statefulSet.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, statefulSet.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteService(service.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilServiceDeleted(f.KubeClient, service.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupBlueprint(f.BackupBlueprint.Name)
				err = framework.WaitUntilBackupBlueprintDeleted(f.StashClient, f.BackupBlueprint.Name)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupConfiguration(statefulSet.ObjectMeta)
				err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, statefulSet.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteRepository(f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilRepositoryDeleted(f.StashClient, f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

			})
			It("Should success auto-backup for the StatefulSet", func() {
				testStatefulSetAutoBackupSucceeded()
			})
		})
		Context("Failure event: ", func() {
			BeforeEach(func() {
				service = f.HeadlessService()
				statefulSet = f.StatefulSetForV1beta1API()
			})
			JustBeforeEach(func() {
				By("Creating service " + service.Name)
				err := f.CreateOrPatchService(service)
				Expect(err).NotTo(HaveOccurred())
			})
			AfterEach(func() {
				err := f.DeleteStatefulSet(statefulSet.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilStatefulSetDeleted(f.KubeClient, statefulSet.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteService(service.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilServiceDeleted(f.KubeClient, service.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupBlueprint(f.BackupBlueprint.Name)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilBackupBlueprintDeleted(f.StashClient, f.BackupBlueprint.Name)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupConfiguration(statefulSet.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, statefulSet.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteRepository(f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilRepositoryDeleted(f.StashClient, f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

			})
			It("Should fail auto-backup for adding inappropriate BackupBlueprint annotation in StatefulSet", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s ", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating BackupBlueprint: %s ", f.BackupBlueprint.Name))
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating StatefulSet: %s/%s", statefulSet.Namespace, statefulSet.Name))
				_, err = f.CreateStatefulSet(statefulSet)
				err = util.WaitUntilStatefulSetReady(f.KubeClient, statefulSet.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Create and Ensure sample data inside running workload")
				err = f.CreateAndEnsureSampleDataInsideWorkload(statefulSet.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Adding auto-backup inappropriate annotations to the StatefulSet: %s/%s", statefulSet.Namespace, statefulSet.Name))
				wrongBackupBlueprintName := "backup-blueprint"
				statefulSet, _, err := apps_util.PatchStatefulSet(f.KubeClient, &statefulSet, func(in *apps.StatefulSet) *apps.StatefulSet {
					in.SetAnnotations(map[string]string{
						v1beta1.KeyBackupBlueprint: wrongBackupBlueprintName,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					})
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the auto-backup annotations has been added successfully")
				Expect(statefulSet.Annotations[v1beta1.KeyBackupBlueprint]).To(Equal(wrongBackupBlueprintName))
				Expect(statefulSet.Annotations[v1beta1.KeyTargetPaths]).To(Equal(framework.TestSourceDataTargetPath))
				Expect(statefulSet.Annotations[v1beta1.KeyVolumeMounts]).To(Equal(framework.TestSourceDataVolumeMount))

				By("Will fail to get respective BackupBlueprint")
				annotations := statefulSet.Annotations
				_, err = f.GetBackupBlueprint(annotations[v1beta1.KeyBackupBlueprint])
				Expect(err).To(HaveOccurred())
			})
			It("Should fail auto-backup for adding inappropriate TargetPath/MountPath annotations in StatefulSet", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s ", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating BackupBlueprint: %s ", f.BackupBlueprint.Name))
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating StatefulSet: %s/%s", statefulSet.Namespace, statefulSet.Name))
				_, err = f.CreateStatefulSet(statefulSet)
				err = util.WaitUntilStatefulSetReady(f.KubeClient, statefulSet.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Create and Ensure sample data inside running workload")
				err = f.CreateAndEnsureSampleDataInsideWorkload(statefulSet.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Adding auto-backup inappropriate annotations to the StatefulSet: %s/%s", statefulSet.Namespace, statefulSet.Name))
				wrongTargetPath := "/source/data-1"
				statefulSet, _, err := apps_util.PatchStatefulSet(f.KubeClient, &statefulSet, func(in *apps.StatefulSet) *apps.StatefulSet {
					in.SetAnnotations(map[string]string{
						v1beta1.KeyBackupBlueprint: f.BackupBlueprint.Name,
						v1beta1.KeyTargetPaths:     wrongTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					})
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the auto-backup annotations has been added successfully")
				Expect(statefulSet.Annotations[v1beta1.KeyBackupBlueprint]).To(Equal(f.BackupBlueprint.Name))
				Expect(statefulSet.Annotations[v1beta1.KeyTargetPaths]).To(Equal(wrongTargetPath))
				Expect(statefulSet.Annotations[v1beta1.KeyVolumeMounts]).To(Equal(framework.TestSourceDataVolumeMount))

				By("Waiting for Repository")
				f.EventuallyRepositoryCreated(f.Namespace()).Should(BeTrue())
				f.Repository, err = f.GetRepository(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupConfiguration")
				f.EventuallyBackupConfigurationCreated(f.Namespace()).Should(BeTrue())
				f.BackupConfig, err = f.GetBackupConfiguration(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession")
				f.EventuallyBackupSessionCreated(f.Namespace()).Should(BeTrue())
				f.BackupSession, err = f.GetBackupSession(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession to be failed")
				f.EventuallyBackupSessionPhase(f.BackupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionFailed))
			})

			It("Should fail auto-backup for adding inappropriate Repository secret in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.Backend.StorageSecretName = ""
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).To(HaveOccurred())
			})
			It("Should fail auto-backup for adding inappropriate Repository backend in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.Backend.GCS = &v1.GCSSpec{}
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating StateFulSet: %s/%s", statefulSet.Namespace, statefulSet.Name))
				_, err = f.CreateStatefulSet(statefulSet)
				err = util.WaitUntilStatefulSetReady(f.KubeClient, statefulSet.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Create and Ensure sample data inside running workload")
				err = f.CreateAndEnsureSampleDataInsideWorkload(statefulSet.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Adding auto-backup specific annotations to the StatefulSet: %s/%s", statefulSet.Namespace, statefulSet.Name))
				err = f.AddAutoBackupAnnotationsToTarget(f.BackupBlueprint.Name, statefulSet.ObjectMeta, apis.KindStatefulSet)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the auto-backup annotations has been added successfully")
				f.EventuallyAutoBackupAnnotationsFound(f.BackupBlueprint.Name, statefulSet.ObjectMeta, apis.KindStatefulSet).Should(BeTrue())

				By("Waiting for Repository")
				f.EventuallyRepositoryCreated(f.Namespace()).Should(BeTrue())
				f.Repository, err = f.GetRepository(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupConfiguration")
				f.EventuallyBackupConfigurationCreated(f.Namespace()).Should(BeTrue())
				f.BackupConfig, err = f.GetBackupConfiguration(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession")
				f.EventuallyBackupSessionCreated(f.Namespace()).Should(BeTrue())
				f.BackupSession, err = f.GetBackupSession(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("waiting for BackupSession to be failed")
				f.EventuallyBackupSessionPhase(f.BackupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionFailed))

			})
			It("Should fail auto-backup for adding inappropriate BackupConfiguration RetentionPolicy in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.RetentionPolicy = v1alpha1.RetentionPolicy{}
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).To(HaveOccurred())
			})
			It("Should fail auto-backup for adding inappropriate BackupConfiguration Schedule in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.Schedule = ""
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).To(HaveOccurred())
			})
		})

	})

	var _ = Describe("DaemonSet", func() {
		var (
			pvc       *core.PersistentVolumeClaim
			daemonset apps.DaemonSet

			testDaemonSetAutoBackupSucceeded = func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s ", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating BackupBlueprint: %s ", f.BackupBlueprint.Name))
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating DaemonSet: %s/%s", daemonset.Namespace, daemonset.Name))
				_, err = f.CreateDaemonSet(daemonset)
				err = util.WaitUntilDaemonSetReady(f.KubeClient, daemonset.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = f.WaitUntilDaemonPodReady(daemonset.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				f.EventuallyPodAccessible(daemonset.ObjectMeta).Should(BeTrue())

				By("Create and Ensure sample data inside running workload")
				err = f.CreateAndEnsureSampleDataInsideWorkload(daemonset.ObjectMeta, apis.KindDaemonSet)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Adding auto-backup specific annotations to the DaemonSet: %s/%s", daemonset.Namespace, daemonset.Name))
				err = f.AddAutoBackupAnnotationsToTarget(f.BackupBlueprint.Name, daemonset.ObjectMeta, apis.KindDaemonSet)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the auto-backup annotations has been added successfully")
				f.EventuallyAutoBackupAnnotationsFound(f.BackupBlueprint.Name, daemonset.ObjectMeta, apis.KindDaemonSet).Should(BeTrue())

				By("Waiting for Repository")
				f.EventuallyRepositoryCreated(f.Namespace()).Should(BeTrue())
				f.Repository, err = f.GetRepository(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupConfiguration")
				f.EventuallyBackupConfigurationCreated(f.Namespace()).Should(BeTrue())
				f.BackupConfig, err = f.GetBackupConfiguration(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession")
				f.EventuallyBackupSessionCreated(f.Namespace()).Should(BeTrue())
				f.BackupSession, err = f.GetBackupSession(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession to be succeeded")
				f.EventuallyBackupSessionPhase(f.BackupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionSucceeded))

				By("Check for repository status to be updated")
				f.EventuallyRepository(&daemonset).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
			}
		)
		BeforeEach(func() {
			f = root.Invoke()
		})
		JustBeforeEach(func() {
			cred = f.SecretForGCSBackend()
			if missing, _ := BeZero().Match(cred); missing {
				Skip("Missing repository credential")
			}
			f.BackupBlueprint = f.BackupBlueprintObj(cred.Name)
		})
		AfterEach(func() {
			err := f.DeleteSecret(cred.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})
		Context("Success event: ", func() {
			BeforeEach(func() {
				pvc = f.GetPersistentVolumeClaim()
				err := f.CreatePersistentVolumeClaim(pvc)
				Expect(err).NotTo(HaveOccurred())
				daemonset = f.DaemonSet(pvc.Name)
			})
			AfterEach(func() {
				err := f.DeleteDaemonSet(daemonset.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilDaemonSetDeleted(f.KubeClient, daemonset.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupBlueprint(f.BackupBlueprint.Name)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilBackupBlueprintDeleted(f.StashClient, f.BackupBlueprint.Name)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupConfiguration(f.BackupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, f.BackupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteRepository(f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilRepositoryDeleted(f.StashClient, f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeletePersistentVolumeClaim(pvc.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

			})
			It("Should success auto-backup for the DaemonSet", func() {
				testDaemonSetAutoBackupSucceeded()
			})
		})
		Context("Failure event: ", func() {
			BeforeEach(func() {
				pvc = f.GetPersistentVolumeClaim()
				err := f.CreatePersistentVolumeClaim(pvc)
				Expect(err).NotTo(HaveOccurred())
				daemonset = f.DaemonSet(pvc.Name)
			})
			AfterEach(func() {
				err := f.DeleteDaemonSet(daemonset.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilDaemonSetDeleted(f.KubeClient, daemonset.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupBlueprint(f.BackupBlueprint.Name)
				err = framework.WaitUntilBackupBlueprintDeleted(f.StashClient, f.BackupBlueprint.Name)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupConfiguration(f.BackupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, f.BackupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteRepository(f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilRepositoryDeleted(f.StashClient, f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeletePersistentVolumeClaim(pvc.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

			})
			It("Should fail auto-backup for adding inappropriate Repository secret in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.Backend.StorageSecretName = ""
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).To(HaveOccurred())
			})
			It("Should fail auto-backup for adding inappropriate Repository backend in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.Backend.GCS = &v1.GCSSpec{}
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating DaemonSet: %s/%s", daemonset.Namespace, daemonset.Name))
				_, err = f.CreateDaemonSet(daemonset)
				err = util.WaitUntilDaemonSetReady(f.KubeClient, daemonset.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = f.WaitUntilDaemonPodReady(daemonset.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				f.EventuallyPodAccessible(daemonset.ObjectMeta).Should(BeTrue())

				By("Create and Ensure sample data inside running workload")
				err = f.CreateAndEnsureSampleDataInsideWorkload(daemonset.ObjectMeta, apis.KindDaemonSet)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Adding auto-backup specific annotations to the DaemonSet: %s/%s", daemonset.Namespace, daemonset.Name))
				err = f.AddAutoBackupAnnotationsToTarget(f.BackupBlueprint.Name, daemonset.ObjectMeta, apis.KindDaemonSet)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the auto-backup annotations has been added successfully")
				f.EventuallyAutoBackupAnnotationsFound(f.BackupBlueprint.Name, daemonset.ObjectMeta, apis.KindDaemonSet).Should(BeTrue())

				By("Waiting for Repository")
				f.EventuallyRepositoryCreated(f.Namespace()).Should(BeTrue())
				f.Repository, err = f.GetRepository(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupConfiguration")
				f.EventuallyBackupConfigurationCreated(f.Namespace()).Should(BeTrue())
				f.BackupConfig, err = f.GetBackupConfiguration(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession")
				f.EventuallyBackupSessionCreated(f.Namespace()).Should(BeTrue())
				f.BackupSession, err = f.GetBackupSession(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("waiting for BackupSession to be failed")
				f.EventuallyBackupSessionPhase(f.BackupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionFailed))

			})
			It("Should fail auto-backup for adding inappropriate BackupConfiguration RetentionPolicy in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.RetentionPolicy = v1alpha1.RetentionPolicy{}
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).To(HaveOccurred())
			})
			It("Should fail auto-backup for adding inappropriate BackupConfiguration Schedule in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.Schedule = ""
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).To(HaveOccurred())
			})

			It("Should fail auto-backup for adding inappropriate BackupBlueprint annotation in DaemonSet", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s ", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating BackupBlueprint: %s ", f.BackupBlueprint.Name))
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating DaemonSet: %s/%s", daemonset.Namespace, daemonset.Name))
				_, err = f.CreateDaemonSet(daemonset)
				err = util.WaitUntilDaemonSetReady(f.KubeClient, daemonset.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = f.WaitUntilDaemonPodReady(daemonset.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				f.EventuallyPodAccessible(daemonset.ObjectMeta).Should(BeTrue())

				By("Create and Ensure sample data inside running workload")
				err = f.CreateAndEnsureSampleDataInsideWorkload(daemonset.ObjectMeta, apis.KindDaemonSet)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Adding auto-backup inappropriate annotations to the DaemonSet: %s/%s", daemonset.Namespace, daemonset.Name))
				wrongBackupBlueprintName := "backup-blueprint"
				daemonset, _, err := apps_util.PatchDaemonSet(f.KubeClient, &daemonset, func(in *apps.DaemonSet) *apps.DaemonSet {
					in.SetAnnotations(map[string]string{
						v1beta1.KeyBackupBlueprint: wrongBackupBlueprintName,
						v1beta1.KeyTargetPaths:     framework.TestSourceDataTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					})
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the auto-backup annotations has been added successfully")
				Expect(daemonset.Annotations[v1beta1.KeyBackupBlueprint]).To(Equal(wrongBackupBlueprintName))
				Expect(daemonset.Annotations[v1beta1.KeyTargetPaths]).To(Equal(framework.TestSourceDataTargetPath))
				Expect(daemonset.Annotations[v1beta1.KeyVolumeMounts]).To(Equal(framework.TestSourceDataVolumeMount))

				By("Will fail to get respective BackupBlueprint")
				annotations := daemonset.Annotations
				_, err = f.GetBackupBlueprint(annotations[v1beta1.KeyBackupBlueprint])
				Expect(err).To(HaveOccurred())

			})
			It("Should fail auto-backup for adding inappropriate TargetPath/MountPath annotations in DaemonSet", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s ", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating BackupBlueprint: %s ", f.BackupBlueprint.Name))
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating DaemonSet: %s/%s", daemonset.Namespace, daemonset.Name))
				_, err = f.CreateDaemonSet(daemonset)
				err = util.WaitUntilDaemonSetReady(f.KubeClient, daemonset.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = f.WaitUntilDaemonPodReady(daemonset.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				f.EventuallyPodAccessible(daemonset.ObjectMeta).Should(BeTrue())

				By("Create and Ensure sample data inside running workload")
				err = f.CreateAndEnsureSampleDataInsideWorkload(daemonset.ObjectMeta, apis.KindDaemonSet)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Adding auto-backup inappropriate annotations to the DaemonSet: %s/%s", daemonset.Namespace, daemonset.Name))
				wrongTargetPath := "/source/data-1"
				daemonset, _, err := apps_util.PatchDaemonSet(f.KubeClient, &daemonset, func(in *apps.DaemonSet) *apps.DaemonSet {
					in.SetAnnotations(map[string]string{
						v1beta1.KeyBackupBlueprint: f.BackupBlueprint.Name,
						v1beta1.KeyTargetPaths:     wrongTargetPath,
						v1beta1.KeyVolumeMounts:    framework.TestSourceDataVolumeMount,
					})
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the auto-backup annotations has been added successfully")
				Expect(daemonset.Annotations[v1beta1.KeyBackupBlueprint]).To(Equal(f.BackupBlueprint.Name))
				Expect(daemonset.Annotations[v1beta1.KeyTargetPaths]).To(Equal(wrongTargetPath))
				Expect(daemonset.Annotations[v1beta1.KeyVolumeMounts]).To(Equal(framework.TestSourceDataVolumeMount))

				By("Waiting for Repository")
				f.EventuallyRepositoryCreated(f.Namespace()).Should(BeTrue())
				f.Repository, err = f.GetRepository(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupConfiguration")
				f.EventuallyBackupConfigurationCreated(f.Namespace()).Should(BeTrue())
				f.BackupConfig, err = f.GetBackupConfiguration(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession")
				f.EventuallyBackupSessionCreated(f.Namespace()).Should(BeTrue())
				f.BackupSession, err = f.GetBackupSession(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession to be failed")
				f.EventuallyBackupSessionPhase(f.BackupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionFailed))
			})

		})
	})

	var _ = Describe("PVC", func() {
		var (
			pvc *core.PersistentVolumeClaim
			pod core.Pod

			testPVCAutoBackupSucceeded = func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s ", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating BackupBlueprint: %s ", f.BackupBlueprint.Name))
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating PVC: %s/%s ", pvc.Namespace, pvc.Name))
				err = f.CreatePersistentVolumeClaim(pvc)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating Pod: %s/%s", pod.Namespace, pod.Name))
				err = f.CreatePod(pod)
				Expect(err).NotTo(HaveOccurred())
				err = core_util.WaitUntilPodRunning(f.KubeClient, pod.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Create and Ensure sample data inside running workload")
				err = f.CreateAndEnsureSampleDataInsideWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Adding auto-backup specific annotations to the PVC: %s/%s", pvc.Namespace, pvc.Name))
				err = f.AddAutoBackupAnnotationsToTarget(f.BackupBlueprint.Name, pvc.ObjectMeta, apis.KindPersistentVolumeClaim)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the auto-backup annotations has been added successfully")
				f.EventuallyAutoBackupAnnotationsFound(f.BackupBlueprint.Name, pvc.ObjectMeta, apis.KindPersistentVolumeClaim).Should(BeTrue())

				By("Waiting for Repository")
				f.EventuallyRepositoryCreated(f.Namespace()).Should(BeTrue())
				f.Repository, err = f.GetRepository(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupConfiguration")
				f.EventuallyBackupConfigurationCreated(f.Namespace()).Should(BeTrue())
				f.BackupConfig, err = f.GetBackupConfiguration(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession")
				f.EventuallyBackupSessionCreated(f.Namespace()).Should(BeTrue())
				f.BackupSession, err = f.GetBackupSession(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession to be succeeded")
				f.EventuallyBackupSessionPhase(f.BackupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionSucceeded))

				By("Waiting for Repository status to be updated")
				f.EventuallyRepository(&pvc).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 1)))
			}
		)
		BeforeEach(func() {
			f = root.Invoke()

			By("Ensure pvc-backup Function exist")
			err := f.VerifyPVCBackupFunction()
			Expect(err).NotTo(HaveOccurred())
			By("Ensure pvc-backup Task exist")
			err = f.VerifyPVCBackupTask()
			Expect(err).NotTo(HaveOccurred())
		})
		JustBeforeEach(func() {
			cred = f.SecretForGCSBackend()
			if missing, _ := BeZero().Match(cred); missing {
				Skip("Missing repository credential")
			}
			f.BackupBlueprint = f.BackupBlueprintObj(cred.Name)
		})
		AfterEach(func() {
			err := f.DeleteSecret(cred.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("Success event: ", func() {
			BeforeEach(func() {
				pvc = f.GetPersistentVolumeClaim()
				pod = f.Pod(pvc.Name)
			})
			JustBeforeEach(func() {
				f.BackupBlueprint.Spec.Task = v1beta1.TaskRef{
					Name: framework.TaskPVCBackup,
				}
			})
			AfterEach(func() {
				err := f.DeletePod(pod.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilPodDeleted(f.KubeClient, pod.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeletePersistentVolumeClaim(pvc.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupBlueprint(f.BackupBlueprint.Name)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilBackupBlueprintDeleted(f.StashClient, f.BackupBlueprint.Name)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupConfiguration(f.BackupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, f.BackupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteRepository(f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilRepositoryDeleted(f.StashClient, f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should success auto-backup for the PVC", func() {
				testPVCAutoBackupSucceeded()
			})
		})
		Context("Failure event: ", func() {
			BeforeEach(func() {
				pvc = f.GetPersistentVolumeClaim()
				pod = f.Pod(pvc.Name)
			})
			JustBeforeEach(func() {
				f.BackupBlueprint.Spec.Task = v1beta1.TaskRef{
					Name: framework.TaskPVCBackup,
				}
			})
			AfterEach(func() {
				err := f.DeletePod(pod.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilPodDeleted(f.KubeClient, pod.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeletePersistentVolumeClaim(pvc.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupBlueprint(f.BackupBlueprint.Name)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilBackupBlueprintDeleted(f.StashClient, f.BackupBlueprint.Name)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteBackupConfiguration(f.BackupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, f.BackupConfig.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				err = f.DeleteRepository(f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
				err = framework.WaitUntilRepositoryDeleted(f.StashClient, f.Repository.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should fail auto-backup for adding inappropriate BackupBlueprint annotation in PVC", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s ", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating BackupBlueprint: %s ", f.BackupBlueprint.Name))
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating PVC: %s/%s ", pvc.Namespace, pvc.Name))
				err = f.CreatePersistentVolumeClaim(pvc)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating Pod: %s/%s", pod.Namespace, pod.Name))
				err = f.CreatePod(pod)
				Expect(err).NotTo(HaveOccurred())
				err = core_util.WaitUntilPodRunning(f.KubeClient, pod.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Create and Ensure sample data inside running workload")
				err = f.CreateAndEnsureSampleDataInsideWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Adding auto-backup inappropriate annotation to the PVC: %s/%s", pvc.Namespace, pvc.Name))
				wrongBackupBlueprintName := "backup-blueprint"
				pvc, _, err = core_util.PatchPVC(f.KubeClient, pvc, func(in *core.PersistentVolumeClaim) *core.PersistentVolumeClaim {
					in.SetAnnotations(map[string]string{
						v1beta1.KeyBackupBlueprint: wrongBackupBlueprintName,
					})
					return in
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the auto-backup annotation have been added successfully")
				Expect(pvc.Annotations[v1beta1.KeyBackupBlueprint]).To(Equal(wrongBackupBlueprintName))

				By("Will fail to get respective BackupBlueprint")
				annotations := pvc.GetAnnotations()
				_, err = f.GetBackupBlueprint(annotations[v1beta1.KeyBackupBlueprint])
				Expect(err).To(HaveOccurred())
			})

			It("Should fail auto-backup for adding inappropriate Repository secret in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s ", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating inappropriate BackupBlueprint: %s", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.Backend.StorageSecretName = ""
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).To(HaveOccurred())
			})
			It("Should fail auto-backup for adding inappropriate Repository backend in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s ", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s ", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.Backend.GCS = &v1.GCSSpec{}
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating PVC: %s/%s ", pvc.Namespace, pvc.Name))
				err = f.CreatePersistentVolumeClaim(pvc)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating Pod: %s/%s", pod.Namespace, pod.Name))
				err = f.CreatePod(pod)
				Expect(err).NotTo(HaveOccurred())
				err = core_util.WaitUntilPodRunning(f.KubeClient, pod.ObjectMeta)
				Expect(err).NotTo(HaveOccurred())

				By("Create and Ensure sample data inside running workload")
				err = f.CreateAndEnsureSampleDataInsideWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Adding auto-backup specific annotations to the PVC: %s/%s", pvc.Namespace, pvc.Name))
				err = f.AddAutoBackupAnnotationsToTarget(f.BackupBlueprint.Name, pvc.ObjectMeta, apis.KindPersistentVolumeClaim)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying that the auto-backup annotations has been added successfully")
				f.EventuallyAutoBackupAnnotationsFound(f.BackupBlueprint.Name, pvc.ObjectMeta, apis.KindPersistentVolumeClaim).Should(BeTrue())

				By("Waiting for Repository")
				f.EventuallyRepositoryCreated(f.Namespace()).Should(BeTrue())
				f.Repository, err = f.GetRepository(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupConfiguration")
				f.EventuallyBackupConfigurationCreated(f.Namespace()).Should(BeTrue())
				f.BackupConfig, err = f.GetBackupConfiguration(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for BackupSession")
				f.EventuallyBackupSessionCreated(f.Namespace()).Should(BeTrue())
				f.BackupSession, err = f.GetBackupSession(f.Namespace())
				Expect(err).NotTo(HaveOccurred())

				By("waiting for BackupSession to be failed")
				f.EventuallyBackupSessionPhase(f.BackupSession.ObjectMeta).Should(Equal(v1beta1.BackupSessionFailed))
			})
			It("Should fail auto-backup for adding inappropriate BackupConfiguration RetentionPolicy in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s ", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating inappropraite BackupBlueprint: %s ", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.RetentionPolicy = v1alpha1.RetentionPolicy{}
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).To(HaveOccurred())
			})
			It("Should fail auto-backup for adding inappropriate BackupConfiguration Schedule in BackupBlueprint", func() {
				By(fmt.Sprintf("Creating storage Secret: %s/%s ", cred.Namespace, cred.Name))
				err := f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating BackupBlueprint: %s ", f.BackupBlueprint.Name))
				f.BackupBlueprint.Spec.Schedule = ""
				_, err = f.CreateBackupBlueprint(f.BackupBlueprint)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
