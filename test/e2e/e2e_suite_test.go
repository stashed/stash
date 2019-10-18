package e2e_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	ka "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"kmodules.xyz/client-go/logs"
	"kmodules.xyz/client-go/tools/cli"
	"kmodules.xyz/client-go/tools/clientcmd"
	"stash.appscode.dev/stash/client/clientset/versioned/scheme"
	_ "stash.appscode.dev/stash/client/clientset/versioned/scheme"
	"stash.appscode.dev/stash/pkg/controller"
	"stash.appscode.dev/stash/test/e2e/framework"
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
	utilruntime.Must(scheme.AddToScheme(clientsetscheme.Scheme))
	utilruntime.Must(scheme.AddToScheme(legacyscheme.Scheme))
	cli.LoggerOptions.Verbosity = "5"

	clientConfig, err := clientcmd.BuildConfigFromContext(options.KubeConfig, options.KubeContext)
	Expect(err).NotTo(HaveOccurred())
	cfg := controller.NewConfig(clientConfig)

	err = options.ApplyTo(cfg)
	Expect(err).NotTo(HaveOccurred())

	kaClient := ka.NewForConfigOrDie(clientConfig)

	root = framework.New(cfg.KubeClient, cfg.StashClient, kaClient, clientConfig, options.StorageClass)
	err = root.CreateTestNamespace()
	Expect(err).NotTo(HaveOccurred())
	By("Using test namespace " + root.Namespace())
})

var _ = AfterSuite(func() {
	By("Deleting namespace: " + root.Namespace())
	err := root.DeleteNamespace(root.Namespace())
	Expect(err).NotTo(HaveOccurred())
})
