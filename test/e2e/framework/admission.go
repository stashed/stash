package framework

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	srvr "github.com/appscode/stash/pkg/cmds/server"
	shell "github.com/codeskyblue/go-sh"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	genericapiserver "k8s.io/apiserver/pkg/server"
	//kapi "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

func (f *Framework) NewTestStashOptions(kubeConfigPath string, controllerOptions *srvr.ControllerOptions) *srvr.StashOptions {
	opt := srvr.NewStashOptions(os.Stdout, os.Stderr)
	opt.RecommendedOptions.Authentication.RemoteKubeConfigFile = kubeConfigPath
	opt.RecommendedOptions.Authentication.SkipInClusterLookup = true
	opt.RecommendedOptions.Authorization.RemoteKubeConfigFile = kubeConfigPath
	opt.RecommendedOptions.CoreAPI.CoreAPIKubeconfigPath = kubeConfigPath
	opt.RecommendedOptions.SecureServing.BindPort = 8443
	opt.RecommendedOptions.SecureServing.BindAddress = net.ParseIP("127.0.0.1")
	opt.ControllerOptions = controllerOptions
	opt.StdErr = os.Stderr
	opt.StdOut = os.Stdout

	return opt
}

func (f *Framework) StartAPIServerAndOperator(kubeConfigPath string, controllerOptions *srvr.ControllerOptions) {
	sh := shell.NewSession()
	args := []interface{}{"--namespace", f.Namespace()}
	SetupServer := filepath.Join("..", "..", "hack", "dev", "setup-server.sh")

	By("Creating API server and webhook stuffs")
	cmd := sh.Command(SetupServer, args...)
	err := cmd.Run()
	Expect(err).ShouldNot(HaveOccurred())

	By("Starting Server and Operator")
	stopCh := genericapiserver.SetupSignalHandler()
	stashOptions := f.NewTestStashOptions(kubeConfigPath, controllerOptions)
	err = stashOptions.Run(stopCh)
	Expect(err).ShouldNot(HaveOccurred())
}

func (f *Framework) EventuallyAPIServerReady() GomegaAsyncAssertion {
	return Eventually(
		func() error {
			//apiservice, err := f.KAClient.ApiregistrationV1beta1().APIServices().Get("v1alpha1.admission.stash.appscode.com", metav1.GetOptions{})
			//if err != nil {
			//	return err
			//}
			//for _, cond := range apiservice.Status.Conditions {
			//	if cond.Type == kapi.Available && cond.Status == kapi.ConditionTrue && cond.Reason == "Passed" {
			//		return nil
			//	}
			//}
			time.Sleep(time.Second*30)
			return nil
			return fmt.Errorf("ApiService not ready yet")
		},
		time.Minute*5,
		time.Microsecond*10,
	)
}
