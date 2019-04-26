package e2e_test

import (
	"flag"
	"path/filepath"

	"github.com/appscode/go/flags"
	logs "github.com/appscode/go/log/golog"
	"github.com/appscode/stash/pkg/cmds/server"
	opt "github.com/appscode/stash/test/e2e/framework"
	"k8s.io/client-go/util/homedir"
)

type E2EOptions struct {
	*server.ExtraOptions
	KubeContext string
	KubeConfig  string
}

var (
	options = &E2EOptions{
		ExtraOptions: server.NewExtraOptions(),
		KubeConfig:   filepath.Join(homedir.HomeDir(), ".kube", "config"),
	}
)

func init() {
	//options.AddGoFlags(flag.CommandLine)
	flag.StringVar(&options.DockerRegistry, "docker-registry", "", "Set Docker Registry")
	flag.StringVar(&options.StashImageTag, "image-tag", "", "Set Stash Image Tag")
	flag.StringVar(&options.KubeConfig, "kubeconfig", options.KubeConfig, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	flag.StringVar(&options.KubeContext, "kube-context", "", "Name of kube context")
	enableLogging()
	flag.Parse()
	opt.DockerRegistry = options.DockerRegistry
	opt.DockerImageTag = options.StashImageTag

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
