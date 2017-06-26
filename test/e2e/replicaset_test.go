package e2e_test

import (
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/util"
	. "github.com/appscode/stash/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

var _ = Describe("ReplicaSet", func() {
	var (
		err    error
		restic sapi.Restic
		rs     extensions.ReplicaSet
	)

	BeforeEach(func() {
		restic = f.Restic()
		rs = f.ReplicaSet()
	})

	Describe("Sidecar added to", func() {
		AfterEach(func() {
			//f.DeleteReplicaSet(rs.ObjectMeta)
			//f.DeleteRestic(restic.ObjectMeta)
		})

		Context("new ReplicaSet", func() {
			FIt(`should backup to "Local" backend`, func() {
				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating ReplicaSet " + rs.Name)
				err = f.CreateReplicaSet(rs)
				Expect(err).NotTo(HaveOccurred())

				f.WaitUntilReplicaSetCondition(rs.ObjectMeta, HaveSidecar(util.StashContainer))
				f.WaitForBackupEvent(restic.Name)
			})
		})

		Context("existing ReplicaSet", func() {
			It(`should backup to "Local" backend`, func() {
				By("Creating ReplicaSet " + rs.Name)
				err = f.CreateReplicaSet(rs)
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
			f.DeleteReplicaSet(rs.ObjectMeta)
		})

		It(`when restic is deleted`, func() {
			By("Creating restic " + restic.Name)
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicaSet " + rs.Name)
			err = f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())

			f.WaitForBackupEvent(restic.Name)

			By("Deleting restic " + restic.Name)
			f.DeleteRestic(restic.ObjectMeta)

			f.WaitUntilReplicaSetCondition(rs.ObjectMeta, HaveSidecar(util.StashContainer))
		})
	})
})
