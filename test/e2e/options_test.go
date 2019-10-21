package e2e_test

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/appscode/go/flags"
	"github.com/appscode/go/log"
	logs "github.com/appscode/go/log/golog"
	"k8s.io/client-go/util/homedir"
	"stash.appscode.dev/stash/pkg/cmds/server"
)

type E2EOptions struct {
	*server.ExtraOptions
	KubeContext  string
	KubeConfig   string
	StorageClass string
}

var (
	options = &E2EOptions{
		ExtraOptions: server.NewExtraOptions(),
		KubeConfig: func() string {
			kubecfg := os.Getenv("KUBECONFIG")
			if kubecfg != "" {
				return kubecfg
			}
			return filepath.Join(homedir.HomeDir(), ".kube", "config")
		}(),
	}
)

func init() {
	flag.StringVar(&options.DockerRegistry, "docker-registry", "", "Set Docker Registry")
	flag.StringVar(&options.StashImageTag, "image-tag", "", "Set Stash Image Tag")
	flag.StringVar(&options.KubeConfig, "kubeconfig", options.KubeConfig, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	flag.StringVar(&options.KubeContext, "kube-context", "", "Name of kube context")
	flag.StringVar(&options.StorageClass, "storageclass", "standard", "Storageclass for PVC")
	enableLogging()
	flag.Parse()
}

func enableLogging() {
	defer func() {
		logs.InitLogs()
		defer logs.FlushLogs()
	}()
	err := flag.Set("logtostderr", "true")
	if err != nil {
		log.Errorln(err)
	}
	logLevelFlag := flag.Lookup("v")
	if logLevelFlag != nil {
		if len(logLevelFlag.Value.String()) > 0 && logLevelFlag.Value.String() != "0" {
			return
		}
	}
	flags.SetLogLevel(2)
}
