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

package framework

import (
	"path/filepath"

	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"

	. "github.com/onsi/gomega" // nolint: staticcheck
	"gomodules.xyz/blobfs"
	"gomodules.xyz/cert/certstore"
	"gomodules.xyz/x/crypto/rand"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ka "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	"kmodules.xyz/client-go/meta"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
)

type Framework struct {
	KubeClient      kubernetes.Interface
	StashClient     cs.Interface
	KAClient        ka.Interface
	catalogClient   appcatalog_cs.Interface
	dmClient        dynamic.Interface
	namespace       string
	CertStore       *certstore.CertStore
	ClientConfig    *rest.Config
	StorageClass    string
	DockerRegistry  string
	SlackWebhookURL string
}

// RootFramework will be used to invoke new Invocation before each test from the individual test packages
var RootFramework *Framework

var TestFailed = false

func New(clientConfig *rest.Config, storageClass, registry, slackWebhook string) *Framework {
	store := certstore.New(blobfs.NewInMemoryFS(), filepath.Join("", "pki"), 0)

	err := store.InitCA()
	Expect(err).NotTo(HaveOccurred())

	kubeClient := kubernetes.NewForConfigOrDie(clientConfig)
	stashClient := cs.NewForConfigOrDie(clientConfig)
	kaClient := ka.NewForConfigOrDie(clientConfig)
	dmClient := dynamic.NewForConfigOrDie(clientConfig)
	catalogClient := appcatalog_cs.NewForConfigOrDie(clientConfig)

	return &Framework{
		KubeClient:      kubeClient,
		StashClient:     stashClient,
		KAClient:        kaClient,
		dmClient:        dmClient,
		catalogClient:   catalogClient,
		namespace:       rand.WithUniqSuffix("test-stash"),
		CertStore:       store,
		ClientConfig:    clientConfig,
		StorageClass:    storageClass,
		DockerRegistry:  registry,
		SlackWebhookURL: slackWebhook,
	}
}

func NewInvocation() *Invocation {
	return RootFramework.Invoke()
}

func (f *Framework) Invoke() *Invocation {
	inv := &Invocation{
		Framework:     f,
		app:           rand.WithUniqSuffix("stash-e2e"),
		testResources: make([]runtime.Object, 0),
	}
	inv.backupNamespace = meta.NameWithSuffix(inv.app, "backup")
	inv.restoreNamespace = meta.NameWithSuffix(inv.app, "restore")
	return inv
}

func (fi *Invocation) AppLabel() string {
	return "app=" + fi.app
}

func (fi *Invocation) App() string {
	return fi.app
}

func (fi *Invocation) BackupNamespace() string {
	return fi.backupNamespace
}

func (fi *Invocation) RestoreNamespace() string {
	return fi.restoreNamespace
}

type Invocation struct {
	*Framework
	app              string
	backupNamespace  string
	restoreNamespace string
	testResources    []runtime.Object
}
