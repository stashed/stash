package e2e_test

import (
	"net"
	"testing"
	"time"

	//	test sources

	"stash.appscode.dev/stash/client/clientset/versioned/scheme"
	_ "stash.appscode.dev/stash/client/clientset/versioned/scheme"
	"stash.appscode.dev/stash/pkg/controller"
	"stash.appscode.dev/stash/test/e2e/framework"
	_ "stash.appscode.dev/stash/test/e2e/volumes"
	_ "stash.appscode.dev/stash/test/e2e/workloads"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
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

	//	test sources
	_ "stash.appscode.dev/stash/test/e2e/auto-backup"
	_ "stash.appscode.dev/stash/test/e2e/volumes"
	_ "stash.appscode.dev/stash/test/e2e/workloads"
)

const (
	TIMEOUT = 20 * time.Minute
)

var (
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
	utilruntime.Must(scheme.AddToScheme(clientsetscheme.Scheme))
	utilruntime.Must(scheme.AddToScheme(legacyscheme.Scheme))
	cli.LoggerOptions.Verbosity = "5"

	clientConfig, err := clientcmd.BuildConfigFromContext(options.KubeConfig, options.KubeContext)
	Expect(err).NotTo(HaveOccurred())
	ctrlConfig := controller.NewConfig(clientConfig)

	err = options.ApplyTo(ctrlConfig)
	Expect(err).NotTo(HaveOccurred())

	kaClient := ka.NewForConfigOrDie(clientConfig)
	dmClient := dynamic.NewForConfigOrDie(clientConfig)

	root = framework.New(ctrlConfig.KubeClient, ctrlConfig.StashClient, kaClient, dmClient, clientConfig, options.StorageClass)
	framework.RootFramework = root
	By("Using test namespace " + root.Namespace())
	err = root.CreateTestNamespace()
	Expect(err).NotTo(HaveOccurred())

	By("Deploy TLS secured Minio Server")
	_, err = root.CreateMinioServer(true, []net.IP{net.ParseIP("127.0.0.1")})
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("Deleting Minio server")
	err := root.DeleteMinioServer()
	Expect(err).NotTo(HaveOccurred())

	By("Deleting namespace: " + root.Namespace())
	err = root.DeleteNamespace(root.Namespace())
	Expect(err).NotTo(HaveOccurred())
})
