/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

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

	"github.com/appscode/go/crypto/rand"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"gomodules.xyz/cert/certstore"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ka "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
)

type Framework struct {
	KubeClient     kubernetes.Interface
	StashClient    cs.Interface
	KAClient       ka.Interface
	catalogClient  appcatalog_cs.Interface
	dmClient       dynamic.Interface
	namespace      string
	CertStore      *certstore.CertStore
	ClientConfig   *rest.Config
	StorageClass   string
	DockerRegistry string
}

// RootFramework will be used to invoke new Invocation before each test from the individual test packages
var RootFramework *Framework

var TestFailed = false

func New(clientConfig *rest.Config, storageClass, registry string) *Framework {
	store, err := certstore.NewCertStore(afero.NewMemMapFs(), filepath.Join("", "pki"))
	Expect(err).NotTo(HaveOccurred())

	err = store.InitCA()
	Expect(err).NotTo(HaveOccurred())

	kubeClient := kubernetes.NewForConfigOrDie(clientConfig)
	stashClient := cs.NewForConfigOrDie(clientConfig)
	kaClient := ka.NewForConfigOrDie(clientConfig)
	dmClient := dynamic.NewForConfigOrDie(clientConfig)
	catalogClient := appcatalog_cs.NewForConfigOrDie(clientConfig)

	return &Framework{
		KubeClient:     kubeClient,
		StashClient:    stashClient,
		KAClient:       kaClient,
		dmClient:       dmClient,
		catalogClient:  catalogClient,
		namespace:      rand.WithUniqSuffix("test-stash"),
		CertStore:      store,
		ClientConfig:   clientConfig,
		StorageClass:   storageClass,
		DockerRegistry: registry,
	}
}
func NewInvocation() *Invocation {
	return RootFramework.Invoke()
}

func (f *Framework) Invoke() *Invocation {
	return &Invocation{
		Framework:     f,
		app:           rand.WithUniqSuffix("stash-e2e"),
		testResources: make([]interface{}, 0),
	}
}

func (fi *Invocation) AppLabel() string {
	return "app=" + fi.app
}

func (fi *Invocation) App() string {
	return fi.app
}

type Invocation struct {
	*Framework
	app           string
	testResources []interface{}
}
