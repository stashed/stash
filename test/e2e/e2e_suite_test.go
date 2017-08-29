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
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	TIMEOUT = 20 * time.Minute
)

var (
	ctrl *controller.Controller
	root *framework.Framework
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
	crdClient := apiextensionsclient.NewForConfigOrDie(config)

	root = framework.New(kubeClient, stashClient)
	err = root.CreateNamespace()
	Expect(err).NotTo(HaveOccurred())
	By("Using test namespace " + root.Namespace())

	ctrl = controller.New(kubeClient, crdClient, stashClient, "canary")
	By("Registering TPR group " + sapi.GroupName)
	err = ctrl.Setup()
	Expect(err).NotTo(HaveOccurred())
	root.EventuallyCRD("restic." + sapi.GroupName).Should(Succeed())

	ctrl.Run()
})

var _ = AfterSuite(func() {
	// root.DeleteNamespace()
})
