package e2e_test

import (
	"testing"
	"time"

	logs "github.com/appscode/go/log/golog"
	"github.com/appscode/kutil/tools/clientcmd"
	api "github.com/appscode/stash/apis/stash"
	"github.com/appscode/stash/client/scheme"
	_ "github.com/appscode/stash/client/scheme"
	"github.com/appscode/stash/pkg/controller"
	"github.com/appscode/stash/pkg/util"
	"github.com/appscode/stash/test/e2e/framework"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
)

const (
	TIMEOUT           = 20 * time.Minute
	TestStashImageTag = "offlineBackupMR"
)

var (
	ctrl *controller.StashController
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
	scheme.AddToScheme(clientsetscheme.Scheme)
	util.LoggerOptions.Verbosity = "5"

	clientConfig, err := clientcmd.BuildConfigFromContext(options.KubeConfig, options.KubeContext)
	Expect(err).NotTo(HaveOccurred())

	ctrlConfig := controller.NewControllerConfig(clientConfig)

	err = options.ApplyTo(ctrlConfig)
	Expect(err).NotTo(HaveOccurred())

	ctrl, err := ctrlConfig.New()
	Expect(err).NotTo(HaveOccurred())

	root = framework.New(ctrlConfig.KubeClient, ctrlConfig.StashClient)
	err = root.CreateNamespace()
	Expect(err).NotTo(HaveOccurred())
	By("Using test namespace " + root.Namespace())

	root.EventuallyCRD("restic." + api.GroupName).Should(Succeed())

	if options.CreateInitConfig {
		By("Creating workload initializer")
		root.CreateInitializerConfiguration(root.InitializerForWorkloads())
	}

	// Now let's start the controller
	go ctrl.RunInformers(nil)
})

var _ = AfterSuite(func() {
	root.DeleteNamespace()
	if options.CreateInitConfig {
		root.DeleteInitializerConfiguration(root.InitializerForWorkloads().ObjectMeta)
	}
})
