/*
Copyright AppsCode Inc. and Contributors

Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e_test

import (
	"net"
	"testing"
	"time"

	//	test sources

	"stash.appscode.dev/apimachinery/client/clientset/versioned/scheme"
	_ "stash.appscode.dev/apimachinery/client/clientset/versioned/scheme"
	"stash.appscode.dev/stash/pkg/controller"
	_ "stash.appscode.dev/stash/test/e2e/backend"
	"stash.appscode.dev/stash/test/e2e/framework"
	_ "stash.appscode.dev/stash/test/e2e/hooks"
	_ "stash.appscode.dev/stash/test/e2e/misc"
	_ "stash.appscode.dev/stash/test/e2e/volumes"
	_ "stash.appscode.dev/stash/test/e2e/workloads"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"kmodules.xyz/client-go/logs"
	"kmodules.xyz/client-go/tools/cli"
	"kmodules.xyz/client-go/tools/clientcmd"
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

	root = framework.New(clientConfig, options.StorageClass, options.DockerRegistry)
	framework.RootFramework = root
	By("Using test namespace " + root.Namespace())
	err = root.CreateTestNamespace()
	Expect(err).NotTo(HaveOccurred())

	By("Deploy TLS secured Minio Server")
	_, err = root.CreateMinioServer(true, []net.IP{net.ParseIP(framework.LocalHostIP)})
	Expect(err).NotTo(HaveOccurred())

	By("Ensure MySQL Addon")
	err = root.EnsureMySQLAddon()
	if !errors.IsAlreadyExists(err) {
		Expect(err).NotTo(HaveOccurred())
	}
})

var _ = AfterSuite(func() {
	if framework.TestFailed {
		root.PrintOperatorLog()
	}

	By("Deleting Minio server")
	err := root.DeleteMinioServer()
	Expect(err).NotTo(HaveOccurred())

	By("Deleting MySQL Addon")
	err = root.EnsureMySQLAddonDeleted()
	if !errors.IsNotFound(err) {
		Expect(err).NotTo(HaveOccurred())
	}

	By("Deleting namespace: " + root.Namespace())
	err = root.DeleteNamespace(root.Namespace())
	Expect(err).NotTo(HaveOccurred())
})
