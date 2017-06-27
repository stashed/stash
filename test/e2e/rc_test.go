package e2e_test

import (
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/util"
	. "github.com/appscode/stash/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

var _ = Describe("ReplicationController", func() {
	var (
		err    error
		restic sapi.Restic
		cred   apiv1.Secret
		rc     apiv1.ReplicationController
	)

	BeforeEach(func() {
		cred = f.SecretForLocalBackend()
		restic = f.Restic()
		restic.Spec.Backend.RepositorySecretName = cred.Name
		rc = f.ReplicationController()
	})

	Describe("Sidecar added to", func() {
		AfterEach(func() {
			f.DeleteReplicationController(rc.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		Context("new ReplicationController", func() {
			It(`should backup to "Local" backend`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating ReplicationController " + rc.Name)
				err = f.CreateReplicationController(rc)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for backup to complete")
				f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *sapi.Restic) int64 {
					return r.Status.BackupCount
				}, BeNumerically(">=", 1)))
			})
		})

		Context("existing ReplicationController", func() {
			It(`should backup to "Local" backend`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating ReplicationController " + rc.Name)
				err = f.CreateReplicationController(rc)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Waiting for sidecar")
				f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

				By("Waiting for backup to complete")
				f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *sapi.Restic) int64 {
					return r.Status.BackupCount
				}, BeNumerically(">=", 1)))
			})
		})
	})

	Describe("Sidecar removed", func() {
		AfterEach(func() {
			f.DeleteReplicationController(rc.ObjectMeta)
			f.DeleteSecret(cred.ObjectMeta)
		})

		It(`when restic is deleted`, func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicationController " + rc.Name)
			err = f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for sidecar")
			f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for backup to complete")
			f.EventuallyRestic(restic.ObjectMeta).Should(WithTransform(func(r *sapi.Restic) int64 {
				return r.Status.BackupCount
			}, BeNumerically(">=", 1)))

			By("Deleting restic " + restic.Name)
			f.DeleteRestic(restic.ObjectMeta)

			f.EventuallyReplicationController(rc.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		})
	})
})
