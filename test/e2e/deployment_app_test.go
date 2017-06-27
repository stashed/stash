package e2e_test

import (
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/util"
	"github.com/appscode/stash/test/e2e/framework"
	. "github.com/appscode/stash/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

var _ = Describe("DeploymentApp", func() {
	var (
		err        error
		f          *framework.Invocation
		restic     sapi.Restic
		cred       apiv1.Secret
		deployment apps.Deployment
	)

	BeforeEach(func() {
		f = root.Invoke()
		cred = f.SecretForLocalBackend()
		restic = f.Restic()
		restic.Spec.Backend.RepositorySecretName = cred.Name
		deployment = f.DeploymentApp()
	})

	Describe("Sidecar added to", func() {
		AfterEach(func() {
			f.DeleteDeploymentApp(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context("new DeploymentApp", func() {
			It(`should backup to "Local" backend`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating DeploymentApp " + deployment.Name)
				err = f.CreateDeploymentApp(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDeploymentApp(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for backup to complete")
				f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *sapi.Restic) int64 {
					return r.Status.BackupCount
				}, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
			})
		})

		Context("existing DeploymentApp", func() {
			It(`should backup to "Local" backend`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating DeploymentApp " + deployment.Name)
				err = f.CreateDeploymentApp(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyDeploymentApp(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for backup to complete")
				f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *sapi.Restic) int64 {
					return r.Status.BackupCount
				}, BeNumerically(">=", 1)))

				By("Waiting for backup event")
				f.EventualEvent(restic.ObjectMeta).Should(WithTransform(f.CountSuccessfulBackups, BeNumerically(">=", 1)))
			})
		})
	})

	Describe("Sidecar removed", func() {
		AfterEach(func() {
			f.DeleteDeploymentApp(deployment.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		It(`when restic is deleted`, func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating DeploymentApp " + deployment.Name)
			err = f.CreateDeploymentApp(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyDeploymentApp(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *sapi.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Deleting restic " + restic.Name)
			f.DeleteRestic(restic.ObjectMeta)

			f.EventuallyDeploymentApp(deployment.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		})
	})
})
