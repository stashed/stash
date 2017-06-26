package e2e_test

import (
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/util"
	. "github.com/appscode/stash/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

var _ = Describe("DaemonSet", func() {
	var (
		err    error
		restic sapi.Restic
		daemon extensions.DaemonSet
	)

	BeforeEach(func() {
		restic = f.Restic()
		daemon = f.DaemonSet()
	})

	Describe("Sidecar added to", func() {
		AfterEach(func() {
			f.DeleteReplicaSet(daemon.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
		})

		Context("new DaemonSet", func() {
			It(`should backup to "Local" backend`, func() {
				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating DaemonSet " + daemon.Name)
				err = f.CreateDaemonSet(daemon)
				Expect(err).NotTo(HaveOccurred())

				f.WaitForBackupEvent(restic.Name)
			})
		})

		Context("existing DaemonSet", func() {
			It(`should backup to "Local" backend`, func() {
				By("Creating DaemonSet " + daemon.Name)
				err = f.CreateDaemonSet(daemon)
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
			f.DeleteReplicaSet(daemon.ObjectMeta)
		})

		It(`when restic is deleted`, func() {
			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating DaemonSet " + daemon.Name)
			err = f.CreateDaemonSet(daemon)
			Expect(err).NotTo(HaveOccurred())

			f.WaitForBackupEvent(restic.Name)

			By("Deleting restic " + restic.Name)
			f.DeleteRestic(restic.ObjectMeta)

			f.EventuallyReplicaSet(daemon.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		})
	})
})
