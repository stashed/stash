package main

import (
	"fmt"
	"time"

	_ "github.com/appscode/restik/api/install"
	"github.com/appscode/restik/pkg"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/util/flag"
	"k8s.io/kubernetes/pkg/util/logs"
	"k8s.io/kubernetes/pkg/util/runtime"
)

var (
	masterURL      string
	kubeconfigPath string
)

func main() {
	pflag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server (overrides any value in kubeconfig)")
	pflag.StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")

	flag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	pflag.VisitAll(func(flag *pflag.Flag) {
		glog.Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})

	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
	if err != nil {
		fmt.Printf("Could not get kubernetes config: %s", err)
		time.Sleep(30 * time.Minute)
		panic(err)
	}
	defer runtime.HandleCrash()

	w := pkg.New(config)
	fmt.Println("Starting tillerc...")
	w.RunAndHold()
}
