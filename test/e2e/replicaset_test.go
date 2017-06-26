package e2e_test

import (
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/util"
	. "github.com/appscode/stash/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

var _ = Describe("ReplicaSet", func() {
	var (
		err    error
		restic sapi.Restic
		cred   apiv1.Secret
		rs     extensions.ReplicaSet
	)

	BeforeEach(func() {
		cred = f.SecretForLocalBackend()
		restic = f.Restic()
		restic.Spec.Backend.RepositorySecretName = cred.Name
		rs = f.ReplicaSet()
	})

	Describe("Sidecar added to", func() {
		AfterEach(func() {
			//f.DeleteReplicaSet(rs.ObjectMeta)
			//f.DeleteRestic(restic.ObjectMeta)
			//f.DeleteSecret(cred.ObjectMeta)
		})

		Context("new ReplicaSet", func() {
			FIt(`should backup to "Local" backend`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating ReplicaSet " + rs.Name)
				err = f.CreateReplicaSet(rs)
				Expect(err).NotTo(HaveOccurred())

				f.EventuallyReplicaSet(rs.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
				f.WaitForBackupEvent(restic.Name)
			})
		})

		Context("existing ReplicaSet", func() {
			It(`should backup to "Local" backend`, func() {
				By("Creating repository Secret " + cred.Name)
				err = f.CreateSecret(cred)
				Expect(err).NotTo(HaveOccurred())

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

			f.EventuallyReplicaSet(rs.ObjectMeta).ShouldNot(HaveSidecar(util.StashContainer))
		})
	})
})
