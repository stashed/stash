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
	*server.ExtraOptions

	KubeContext        string
	KubeConfig         string
	EnableWebhook      bool
	SelfHostedOperator bool
}

var (
	options = &E2EOptions{
		ExtraOptions:       server.NewExtraOptions(),
		KubeConfig:         filepath.Join(homedir.HomeDir(), ".kube", "config"),
		EnableWebhook:      true,
		SelfHostedOperator: false,
	}
)

func init() {
	options.StashImageTag = TestStashImageTag
	options.AddGoFlags(flag.CommandLine)
	flag.StringVar(&options.KubeConfig, "kubeconfig", options.KubeConfig, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	flag.StringVar(&options.KubeContext, "kube-context", "", "Name of kube context")
	flag.BoolVar(&options.EnableWebhook, "webhook", options.EnableWebhook, "Enable Mutating and Validating Webhook")
	flag.BoolVar(&options.SelfHostedOperator, "selfhosted-operator", options.SelfHostedOperator, "Run test in self hosted operator mode")
	enableLogging()
	flag.Parse()

	options.EnableMutatingWebhook = options.EnableWebhook
	options.EnableValidatingWebhook = options.EnableWebhook
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
