package e2e_test

import (
	"path/filepath"
	"testing"
	"time"

	logs "github.com/appscode/log/golog"
	sapi "github.com/appscode/stash/api"
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

const (
	TIMEOUT = 20 * time.Minute
)

var (
	ctrl *controller.Controller
	f    *framework.Framework
)

func TestE2e(t *testing.T) {
	logs.InitLogs()
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(TIMEOUT)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "e2e Suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {
	userHome, err := homedir.Dir()
	Expect(err).NotTo(HaveOccurred())

	kubeconfigPath := filepath.Join(userHome, ".kube/config")
	By("Using kubeconfig from " + kubeconfigPath)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	Expect(err).NotTo(HaveOccurred())

	kubeClient := clientset.NewForConfigOrDie(config)
	stashClient := rcs.NewForConfigOrDie(config)

	f = framework.New(kubeClient, stashClient)
	err = f.CreateNamespace()
	Expect(err).NotTo(HaveOccurred())
	By("Using test namespace " + f.Namespace())

	ctrl = controller.New(kubeClient, stashClient, "canary")
	By("Registering TPR group " + sapi.GroupName)
	err = ctrl.Setup()
	Expect(err).NotTo(HaveOccurred())
	f.EventuallyTPR("restic." + sapi.GroupName).Should(Succeed())

	ctrl.Run()
})

var _ = AfterSuite(func() {
	// f.DeleteNamespace()
})
