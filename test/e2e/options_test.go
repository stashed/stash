/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"stash.appscode.dev/stash/pkg/cmds/server"

	"gomodules.xyz/logs"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
)

type E2EOptions struct {
	*server.ExtraOptions
	KubeContext     string
	KubeConfig      string
	StorageClass    string
	SlackWebhookURL string
}

var options = &E2EOptions{
	ExtraOptions: server.NewExtraOptions(),
	KubeConfig: func() string {
		kubecfg := os.Getenv("KUBECONFIG")
		if kubecfg != "" {
			return kubecfg
		}
		return filepath.Join(homedir.HomeDir(), ".kube", "config")
	}(),
}

// xref: https://github.com/onsi/ginkgo/issues/602#issuecomment-559421839
func TestMain(m *testing.M) {
	flag.StringVar(&options.DockerRegistry, "docker-registry", "", "Set Docker Registry")
	flag.StringVar(&options.StashImageTag, "image-tag", "", "Set Stash Image Tag")
	flag.StringVar(&options.KubeConfig, "kubeconfig", options.KubeConfig, "Path to kubeconfig file with authorization information (the master location is set by the master flag).")
	flag.StringVar(&options.KubeContext, "kube-context", "", "Name of kube context")
	flag.StringVar(&options.StorageClass, "storageclass", "standard", "Storageclass for PVC")
	flag.StringVar(&options.SlackWebhookURL, "slack-webhook", "", "URL of the Slack webhook")

	enableLogging()
	flag.Parse()
	os.Exit(m.Run())
}

func enableLogging() {
	defer func() {
		logs.InitLogs()
		defer logs.FlushLogs()
	}()
	err := flag.Set("logtostderr", "true")
	if err != nil {
		klog.Errorln(err)
	}
	logLevelFlag := flag.Lookup("v")
	if logLevelFlag != nil {
		if len(logLevelFlag.Value.String()) > 0 && logLevelFlag.Value.String() != "0" {
			return
		}
	}
	_ = flag.Set("v", "2")
}
