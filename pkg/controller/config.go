package controller

import (
	"time"

	"github.com/appscode/kutil/discovery"
	cs "github.com/appscode/stash/client/clientset/versioned"
	stashinformers "github.com/appscode/stash/client/informers/externalversions"
	"github.com/appscode/stash/pkg/eventer"
	core "k8s.io/api/core/v1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	mutatingWebhook   = "admission.stash.appscode.com"
	validatingWebhook = "admission.stash.appscode.com"
)

type config struct {
	EnableRBAC     bool
	StashImageTag  string
	DockerRegistry string
	MaxNumRequeues int
	NumThreads     int
	ResyncPeriod   time.Duration
}

type Config struct {
	config

	ClientConfig *rest.Config
	KubeClient   kubernetes.Interface
	StashClient  cs.Interface
	CRDClient    crd_cs.ApiextensionsV1beta1Interface
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
		opt.IncludeUninitialized = true
	}
	ctrl := &StashController{
		config:               c.config,
		clientConfig:         c.ClientConfig,
		kubeClient:           c.KubeClient,
		stashClient:          c.StashClient,
		crdClient:            c.CRDClient,
		kubeInformerFactory:  informers.NewFilteredSharedInformerFactory(c.KubeClient, c.ResyncPeriod, core.NamespaceAll, tweakListOptions),
		stashInformerFactory: stashinformers.NewSharedInformerFactory(c.StashClient, c.ResyncPeriod),
		recorder:             eventer.NewEventRecorder(c.KubeClient, "stash-controller"),
	}

	if err := ctrl.ensureCustomResourceDefinitions(); err != nil {
		return nil, err
	}
	if err := ctrl.UpdateWebhookCABundle(); err != nil {
		return nil, err
	}
	if ctrl.EnableRBAC {
		if err := ctrl.ensureSidecarClusterRole(); err != nil {
			return nil, err
		}
	}

	ctrl.initNamespaceWatcher()
	ctrl.initResticWatcher()
	ctrl.initRecoveryWatcher()
	ctrl.initRepositoryWatcher()
	ctrl.initDeploymentWatcher()
	ctrl.initDaemonSetWatcher()
	ctrl.initStatefulSetWatcher()
	ctrl.initRCWatcher()
	ctrl.initReplicaSetWatcher()
	ctrl.initJobWatcher()

	return ctrl, nil
}
