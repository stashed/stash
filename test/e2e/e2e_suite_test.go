package e2e_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	logs "github.com/appscode/go/log/golog"
	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/client/clientset/versioned/scheme"
	_ "github.com/appscode/stash/client/clientset/versioned/scheme"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/test/e2e/framework"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/discovery"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/homedir"
	ka "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	discovery_util "kmodules.xyz/client-go/discovery"
	"kmodules.xyz/client-go/tools/cli"
	"kmodules.xyz/client-go/tools/clientcmd"
)

const (
	TIMEOUT = 20 * time.Minute
)

var (
	ctrl         *controller.StashController
	root         *framework.Framework
	storageClass = "standard"
)

func TestE2e(t *testing.T) {
	logs.InitLogs()
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(TIMEOUT)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "e2e Suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {
	scheme.AddToScheme(clientsetscheme.Scheme)
	scheme.AddToScheme(legacyscheme.Scheme)
	cli.LoggerOptions.Verbosity = "5"

	clientConfig, err := clientcmd.BuildConfigFromContext(options.KubeConfig, options.KubeContext)
	Expect(err).NotTo(HaveOccurred())
	ctrlConfig := controller.NewConfig(clientConfig)

	discClient, err := discovery.NewDiscoveryClientForConfig(clientConfig)
	serverVersion, err := discovery_util.GetBaseVersion(discClient)
	Expect(err).NotTo(HaveOccurred())
	if strings.Compare(serverVersion, "1.11") >= 0 {
		apis.EnableStatusSubresource = true
	}
	err = options.ApplyTo(ctrlConfig)
	Expect(err).NotTo(HaveOccurred())

	kaClient := ka.NewForConfigOrDie(clientConfig)
	framework.StashProjectRoot = filepath.Join(homedir.HomeDir(), "go", "src", "github.com", "appscode", "stash")

	root = framework.New(ctrlConfig.KubeClient, ctrlConfig.StashClient, kaClient, clientConfig, storageClass)
	err = root.CreateTestNamespace()
	Expect(err).NotTo(HaveOccurred())
	By("Using test namespace " + root.Namespace())

	By("Starting the Stash Operator")
	root.InstallStashOperator(options.KubeConfig, options.ExtraOptions)
})

var _ = AfterSuite(func() {
	By("Deleting Stash Operator")
	root.UninstallStashOperator()
	By("Deleting namespace: " + root.Namespace())
	err := root.DeleteNamespace(root.Namespace())
	Expect(err).NotTo(HaveOccurred())
})
