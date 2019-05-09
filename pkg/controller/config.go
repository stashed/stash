package controller

import (
	"time"

	core "k8s.io/api/core/v1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
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
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	stashinformers "stash.appscode.dev/stash/client/informers/externalversions"
	"stash.appscode.dev/stash/pkg/eventer"
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
	CRDClient        crd_cs.ApiextensionsV1beta1Interface
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

	if err := ctrl.ensureCustomResourceDefinitions(); err != nil {
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

	if err := ctrl.ensureSidecarClusterRole(); err != nil {
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
	ctrl.initBackupSessionWatcher()
	ctrl.initRestoreSessionWatcher()

	return ctrl, nil
}
