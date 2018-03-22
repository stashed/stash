package e2e_test

import (
	"flag"
	"path/filepath"

	"github.com/appscode/go/flags"
	logs "github.com/appscode/go/log/golog"
	"github.com/appscode/stash/pkg/cmds/server"
	"k8s.io/client-go/util/homedir"
)

type E2EOptions struct {
	*server.ControllerOptions

	KubeContext    string
	KubeConfig     string
	StartAPIServer bool
}

var (
	options = &E2EOptions{
		ControllerOptions: server.NewControllerOptions(),
		KubeConfig:        filepath.Join(homedir.HomeDir(), ".kube", "config"),
		StartAPIServer:    false,
	}
)

func init() {
	options.StashImageTag = TestStashImageTag
	options.AddGoFlags(flag.CommandLine)
	flag.StringVar(&options.KubeConfig, "kubeconfig", "", "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	flag.StringVar(&options.KubeContext, "kube-context", "", "Name of kube context")
	flag.BoolVar(&options.StartAPIServer, "webhook", options.StartAPIServer, "Start API server for webhook")
	enableLogging()
	flag.Parse()
}

func enableLogging() {
	defer func() {
		logs.InitLogs()
		defer logs.FlushLogs()
	}()
	flag.Set("logtostderr", "true")
	logLevelFlag := flag.Lookup("v")
	if logLevelFlag != nil {
		if len(logLevelFlag.Value.String()) > 0 && logLevelFlag.Value.String() != "0" {
			return
		}
	}
	flags.SetLogLevel(2)
}
