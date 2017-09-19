package e2e_test

import (
	"path/filepath"
	"testing"
	"time"

	logs "github.com/appscode/go/log/golog"
	sapi "github.com/appscode/stash/apis/stash"
	"github.com/appscode/stash/client/internalclientset/typed/stash/internalversion"
	_ "github.com/appscode/stash/client/scheme"
	scs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/test/e2e/framework"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
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
	kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube/config")
	By("Using kubeconfig from " + kubeconfigPath)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	Expect(err).NotTo(HaveOccurred())

	kubeClient := clientset.NewForConfigOrDie(config)
	stashClient := scs.NewForConfigOrDie(config)
	internalClient := internalversion.NewForConfigOrDie(config)
	crdClient := apiextensionsclient.NewForConfigOrDie(config)

	root = framework.New(kubeClient, stashClient)
	err = root.CreateNamespace()
	Expect(err).NotTo(HaveOccurred())
	By("Using test namespace " + root.Namespace())

	ctrl = controller.New(kubeClient, crdClient, internalClient, "canary", 5*time.Minute)
	By("Registering CRD group " + sapi.GroupName)
	err = ctrl.Setup()
	Expect(err).NotTo(HaveOccurred())
	root.EventuallyCRD("restic." + sapi.GroupName).Should(Succeed())

	ctrl.Run()
})

var _ = AfterSuite(func() {
	// root.DeleteNamespace()
})
