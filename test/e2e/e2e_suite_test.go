package e2e_test

import (
	"path/filepath"
	"testing"

	rcs "github.com/appscode/stash/client/clientset"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/test/e2e/framework"
	"github.com/mitchellh/go-homedir"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	ctrl *controller.Controller
	f    *framework.Framework
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "E2e Suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {
	userHome, err := homedir.Dir()
	Expect(err).NotTo(HaveOccurred())

	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(userHome, ".kube/config"))
	Expect(err).NotTo(HaveOccurred())

	kubeClient := clientset.NewForConfigOrDie(config)
	stashClient := rcs.NewForConfigOrDie(config)

	f = framework.New(kubeClient, stashClient)
	err = f.CreateNamespace()
	Expect(err).NotTo(HaveOccurred())

	ctrl = controller.New(kubeClient, stashClient, "canary")
	err = ctrl.Setup()
	Expect(err).NotTo(HaveOccurred())

	ctrl.Run()
})

var _ = AfterSuite(func() {
	f.DeleteNamespace()
})
