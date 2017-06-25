package e2e_test

import (
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/util"
	"github.com/appscode/stash/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

var _ = Describe("DeploymentExtension", func() {
	var (
		err        error
		restic     sapi.Restic
		deployment extensions.Deployment
	)

	BeforeEach(func() {
		restic = f.Restic()
		deployment = f.DeploymentExtension()
	})

	Describe("Sidecar added to", func() {
		AfterEach(func() {
			f.DeleteDeploymentExtension(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
		})

		Context("new Deployment", func() {
			It(`should backup to "Local"" backend`, func() {
				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating Deployment " + deployment.Name)
				err = f.CreateDeploymentExtension(deployment)
				Expect(err).NotTo(HaveOccurred())

				f.WaitForBackupEvent(restic.Name)
			})
		})

		Context("existing Deployment", func() {
			It(`should backup to "Local"" backend`, func() {
				By("Creating Deployment " + deployment.Name)
				err = f.CreateDeploymentExtension(deployment)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				f.WaitForBackupEvent(restic.Name)
			})
		})
	})

	Describe("Sidecar removed", func() {
		AfterEach(func() {
			f.DeleteDeploymentExtension(deployment.ObjectMeta)
		})

		It(`when restic is deleted`, func() {
			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			err = f.CreateDeploymentExtension(deployment)
			Expect(err).NotTo(HaveOccurred())

			f.WaitForBackupEvent(restic.Name)

			By("Deleting restic " + restic.Name)
			f.DeleteRestic(restic.ObjectMeta)

			f.WaitUntilDeploymentExtensionCondition(deployment.ObjectMeta, matcher.HaveSidecar(util.StashContainer))
		})
	})
})
