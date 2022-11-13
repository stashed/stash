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

package controller

import (
	"fmt"
	"time"

	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stashinformers "stash.appscode.dev/apimachinery/client/informers/externalversions"
	"stash.appscode.dev/apimachinery/pkg/docker"
	"stash.appscode.dev/stash/pkg/eventer"
	"stash.appscode.dev/stash/pkg/util"

	auditlib "go.bytebuilders.dev/audit/lib"
	proxyserver "go.bytebuilders.dev/license-proxyserver/apis/proxyserver/v1alpha1"
	licenseapi "go.bytebuilders.dev/license-verifier/apis/licenses/v1alpha1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	reg_util "kmodules.xyz/client-go/admissionregistration/v1"
	"kmodules.xyz/client-go/discovery"
	"kmodules.xyz/client-go/tools/clusterid"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
	oc_cs "kmodules.xyz/openshift/client/clientset/versioned"
	oc_informers "kmodules.xyz/openshift/client/informers/externalversions"
)

const (
	mutatingWebhook   = "admission.stash.appscode.com"
	validatingWebhook = "admission.stash.appscode.com"
)

type config struct {
	LicenseFile             string
	License                 licenseapi.License
	LicenseApiService       string
	StashImage              string
	StashImageTag           string
	DockerRegistry          string
	ImagePullSecrets        []string
	MaxNumRequeues          int
	NumThreads              int
	ResyncPeriod            time.Duration
	EnableValidatingWebhook bool
	EnableMutatingWebhook   bool
	CronJobPSPNames         []string
	BackupJobPSPNames       []string
	RestoreJobPSPNames      []string
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

func (c Config) LicenseProvided() bool {
	if c.LicenseFile != "" {
		return true
	}

	ok, _ := discovery.HasGVK(
		c.KubeClient.Discovery(),
		proxyserver.SchemeGroupVersion.String(),
		proxyserver.ResourceKindLicenseRequest)
	return ok
}

func (c *Config) New() (*StashController, error) {
	if err := discovery.IsDefaultSupportedVersion(c.KubeClient); err != nil {
		return nil, err
	}

	mapper, err := discovery.NewDynamicResourceMapper(c.ClientConfig)
	if err != nil {
		return nil, err
	}

	informerFactory := informers.NewSharedInformerFactoryWithOptions(c.KubeClient, c.ResyncPeriod)

	// audit event publisher
	// WARNING: https://stackoverflow.com/a/46275411/244009
	var auditor *auditlib.EventPublisher
	if c.LicenseProvided() && !c.License.DisableAnalytics() {
		cmeta, err := clusterid.ClusterMetadata(c.KubeClient.CoreV1().Namespaces())
		if err != nil {
			return nil, fmt.Errorf("failed to extract cluster metadata, reason: %v", err)
		}
		fn := auditlib.BillingEventCreator{
			Mapper:          mapper,
			ClusterMetadata: cmeta,
		}
		auditor = auditlib.NewResilientEventPublisher(func() (*auditlib.NatsConfig, error) {
			return auditlib.NewNatsConfig(c.ClientConfig, cmeta.UID, c.LicenseFile)
		}, mapper, fn.CreateEvent)
		err = auditor.SetupSiteInfoPublisher(c.ClientConfig, c.KubeClient, informerFactory)
		if err != nil {
			return nil, fmt.Errorf("failed to setup site info publisher, reason: %v", err)
		}
	}

	ctrl := &StashController{
		config:               c.config,
		clientConfig:         c.ClientConfig,
		kubeClient:           c.KubeClient,
		ocClient:             c.OcClient,
		stashClient:          c.StashClient,
		crdClient:            c.CRDClient,
		appCatalogClient:     c.AppCatalogClient,
		kubeInformerFactory:  informerFactory,
		stashInformerFactory: stashinformers.NewSharedInformerFactory(c.StashClient, c.ResyncPeriod),
		ocInformerFactory:    oc_informers.NewSharedInformerFactory(c.OcClient, c.ResyncPeriod),
		recorder:             eventer.NewEventRecorder(c.KubeClient, "stash-operator"),
		mapper:               mapper,
		auditor:              auditor,
	}

	// ensure default functions
	image := docker.Docker{
		Registry: ctrl.DockerRegistry,
		Image:    ctrl.StashImage,
		Tag:      ctrl.StashImageTag,
	}
	err = util.EnsureDefaultFunctions(ctrl.stashClient, image)
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

	// init workload watchers
	ctrl.initDeploymentWatcher()
	ctrl.initDaemonSetWatcher()
	ctrl.initStatefulSetWatcher()
	ctrl.initDeploymentConfigWatcher()

	ctrl.initJobWatcher()

	// init v1alpha1 resources watcher
	ctrl.initRepositoryWatcher()

	// init v1beta1 resources watcher
	ctrl.initBackupConfigurationWatcher()
	ctrl.initBackupSessionWatcher()
	ctrl.initRestoreSessionWatcher()

	if auditor != nil {
		if err := auditor.SetupSiteInfoPublisher(ctrl.clientConfig, ctrl.kubeClient, ctrl.kubeInformerFactory); err != nil {
			return nil, err
		}
	}

	return ctrl, nil
}
