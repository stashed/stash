package e2e_test

import (
	"flag"
	"path/filepath"

	"github.com/appscode/stash/pkg/cmds/server"
	"k8s.io/client-go/util/homedir"
)

type E2EOptions struct {
	*server.ExtraOptions
	KubeContext        string
	KubeConfig         string
}

var (
	options = &E2EOptions{
		ExtraOptions:       server.NewExtraOptions(),
		KubeConfig:         filepath.Join(homedir.HomeDir(), ".kube", "config"),
	}
)

func init() {

	flag.StringVar(&options.DockerRegistry, "docker-registry", "", "Set Docker Registry")
	flag.StringVar(&options.StashImageTag, "image-tag", "", "Set Stash Image Tag")
	flag.StringVar(&options.KubeConfig, "kubeconfig", options.KubeConfig, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	flag.StringVar(&options.KubeContext, "kube-context", "", "Name of kube context")

}

