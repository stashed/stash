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

package controller

import (
	"time"

	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stashinformers "stash.appscode.dev/apimachinery/client/informers/externalversions"
	"stash.appscode.dev/stash/pkg/eventer"
	stash_rbac "stash.appscode.dev/stash/pkg/rbac"
	"stash.appscode.dev/stash/pkg/util"

	core "k8s.io/api/core/v1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	reg_util "kmodules.xyz/client-go/admissionregistration/v1beta1"
	"kmodules.xyz/client-go/discovery"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
	appcatalog_informers "kmodules.xyz/custom-resources/client/informers/externalversions"
	oc_cs "kmodules.xyz/openshift/client/clientset/versioned"
	oc_informers "kmodules.xyz/openshift/client/informers/externalversions"
)

const (
	mutatingWebhook   = "admission.stash.appscode.com"
	validatingWebhook = "admission.stash.appscode.com"
)

type config struct {
	StashImageTag           string
	DockerRegistry          string
	MaxNumRequeues          int
	NumThreads              int
	ResyncPeriod            time.Duration
	EnableValidatingWebhook bool
	EnableMutatingWebhook   bool
}

type Config struct {
	config

	ClientConfig     *rest.Config
	KubeClient       kubernetes.Interface
	OcClient         oc_cs.Interface
	StashClient      cs.Interface
	CRDClient        crd_cs.Interface
	AppCatalogClient appcatalog_cs.Interface
}

func NewConfig(clientConfig *rest.Config) *Config {
	return &Config{
		ClientConfig: clientConfig,
	}
}

func (c *Config) New() (*StashController, error) {
	if err := discovery.IsDefaultSupportedVersion(c.KubeClient); err != nil {
		return nil, err
	}

	tweakListOptions := func(opt *metav1.ListOptions) {
	}
	ctrl := &StashController{
		config:           c.config,
		clientConfig:     c.ClientConfig,
		kubeClient:       c.KubeClient,
		ocClient:         c.OcClient,
		stashClient:      c.StashClient,
		crdClient:        c.CRDClient,
		appCatalogClient: c.AppCatalogClient,
		kubeInformerFactory: informers.NewSharedInformerFactoryWithOptions(
			c.KubeClient,
			c.ResyncPeriod,
			informers.WithNamespace(core.NamespaceAll),
			informers.WithTweakListOptions(tweakListOptions)),
		stashInformerFactory:      stashinformers.NewSharedInformerFactory(c.StashClient, c.ResyncPeriod),
		appCatalogInformerFactory: appcatalog_informers.NewSharedInformerFactory(c.AppCatalogClient, c.ResyncPeriod),
		ocInformerFactory:         oc_informers.NewSharedInformerFactory(c.OcClient, c.ResyncPeriod),
		recorder:                  eventer.NewEventRecorder(c.KubeClient, "stash-operator"),
	}

	// register CRDs
	if err := ctrl.ensureCustomResourceDefinitions(); err != nil {
		return nil, err
	}

	// ensure default functions
	err := util.EnsureDefaultFunctions(ctrl.stashClient, ctrl.DockerRegistry, ctrl.StashImageTag)
	if err != nil {
		return nil, err
	}

	// ensure default tasks
	err = util.EnsureDefaultTasks(ctrl.stashClient)
	if err != nil {
		return nil, err
	}

	if c.EnableMutatingWebhook {
		if err := reg_util.UpdateMutatingWebhookCABundle(c.ClientConfig, mutatingWebhook); err != nil {
			return nil, err
		}
	}
	if c.EnableValidatingWebhook {
		if err := reg_util.UpdateValidatingWebhookCABundle(c.ClientConfig, validatingWebhook); err != nil {
			return nil, err
		}
	}

	if err := stash_rbac.EnsureSidecarClusterRole(c.KubeClient); err != nil {
		return nil, err
	}

	ctrl.initNamespaceWatcher()

	// init workload watchers
	ctrl.initDeploymentWatcher()
	ctrl.initDaemonSetWatcher()
	ctrl.initStatefulSetWatcher()
	ctrl.initRCWatcher()
	ctrl.initReplicaSetWatcher()
	ctrl.initDeploymentConfigWatcher()

	ctrl.initPVCWatcher()
	ctrl.initJobWatcher()
	ctrl.initAppBindingWatcher()

	// init v1alpha1 resources watcher
	ctrl.initResticWatcher()
	ctrl.initRecoveryWatcher()
	ctrl.initRepositoryWatcher()

	// init v1beta1 resources watcher
	ctrl.initBackupConfigurationWatcher()
	ctrl.initBackupBatchWatcher()
	ctrl.initBackupSessionWatcher()
	ctrl.initRestoreSessionWatcher()

	return ctrl, nil
}
