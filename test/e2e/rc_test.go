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
		rc     apiv1.ReplicationController
	)

	BeforeEach(func() {
		restic = f.Restic()
		rc = f.ReplicationController()
	})

	Describe("Sidecar added to", func() {
		AfterEach(func() {
			f.DeleteReplicationController(rc.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
		})

		Context("new rc", func() {
			It(`should backup to "Local" backend`, func() {
				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating rc " + rc.Name)
				err = f.CreateReplicationController(rc)
				Expect(err).NotTo(HaveOccurred())

				f.WaitForBackupEvent(restic.Name)
			})
		})

		Context("existing rc", func() {
			It(`should backup to "Local" backend`, func() {
				By("Creating rc " + rc.Name)
				err = f.CreateReplicationController(rc)
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
			f.DeleteReplicationController(rc.ObjectMeta)
		})

		It(`when restic is deleted`, func() {
			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating rc " + rc.Name)
			err = f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())

			f.WaitForBackupEvent(restic.Name)

			By("Deleting restic " + restic.Name)
			f.DeleteRestic(restic.ObjectMeta)

			f.WaitUntilReplicationControllerCondition(rc.ObjectMeta, HaveSidecar(util.StashContainer))
		})
	})
})
