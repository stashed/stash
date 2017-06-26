package e2e_test

import (
	sapi "github.com/appscode/stash/api"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

var _ = Describe("StatefulSet", func() {
	var (
		err    error
		restic sapi.Restic
		svc    apiv1.Service
		ss     apps.StatefulSet
	)

	BeforeEach(func() {
		restic = f.Restic()
		svc = f.HeadlessService()
		ss = f.StatefulSet(restic)
	})

	Describe("Sidecar added to", func() {
		AfterEach(func() {
			f.DeleteStatefulSet(ss.ObjectMeta)
			f.DeleteService(svc.ObjectMeta)
			f.DeleteRestic(restic.ObjectMeta)
		})

		Context("new StatefulSet", func() {
			It(`should backup to "Local" backend`, func() {
				By("Creating restic " + restic.Name)
				err = f.CreateRestic(restic)
				Expect(err).NotTo(HaveOccurred())

				By("Creating service " + svc.Name)
				err = f.CreateService(svc)
				Expect(err).NotTo(HaveOccurred())

				By("Creating StatefulSet " + ss.Name)
				err = f.CreateStatefulSet(ss)
				Expect(err).NotTo(HaveOccurred())

				f.WaitForBackupEvent(restic.Name)
			})
		})
	})
})
