package e2e_test

import (
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/util"
	. "github.com/appscode/stash/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

var _ = Describe("DeploymentApp", func() {
	var (
		err        error
		restic     sapi.Restic
		deployment apps.Deployment
	)

	BeforeEach(func() {
		restic = f.Restic()
		deployment = f.DeploymentApp()
	})

	Describe("Sidecar added to", func() {
		AfterEach(func() {
			f.DeleteDeploymentApp(deployment.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
		})

		Context("new Deployment", func() {
			It(`should backup to "Local"" backend`, func() {
				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating Deployment " + deployment.Name)
				err = f.CreateDeploymentApp(deployment)
				Expect(err).NotTo(HaveOccurred())

				f.WaitForBackupEvent(restic.Name)
			})
		})

		Context("existing Deployment", func() {
			It(`should backup to "Local"" backend`, func() {
				By("Creating Deployment " + deployment.Name)
				err = f.CreateDeploymentApp(deployment)
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
			f.DeleteDeploymentApp(deployment.ObjectMeta)
		})

		It(`when restic is deleted`, func() {
			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			err = f.CreateDeploymentApp(deployment)
			Expect(err).NotTo(HaveOccurred())

			f.WaitForBackupEvent(restic.Name)

			By("Deleting restic " + restic.Name)
			f.DeleteRestic(restic.ObjectMeta)

			f.WaitUntilDeploymentAppCondition(deployment.ObjectMeta, HaveSidecar(util.StashContainer))
		})
	})
})
