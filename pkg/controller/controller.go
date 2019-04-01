package controller

import (
	"fmt"

	"github.com/appscode/go/log"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	cs "github.com/appscode/stash/client/clientset/versioned"
	stashinformers "github.com/appscode/stash/client/informers/externalversions"
	stash_listers "github.com/appscode/stash/client/listers/stash/v1alpha1"
	stash_listers_v1beta1 "github.com/appscode/stash/client/listers/stash/v1beta1"
	"github.com/golang/glog"
	crd_api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	apps_listers "k8s.io/client-go/listers/apps/v1"
	batch_listers "k8s.io/client-go/listers/batch/v1"
	core_listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	reg_util "kmodules.xyz/client-go/admissionregistration/v1beta1"
	crdutils "kmodules.xyz/client-go/apiextensions/v1beta1"
	"kmodules.xyz/client-go/tools/queue"
	appCatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
	appcatalog_informers "kmodules.xyz/custom-resources/client/informers/externalversions"
	abListers "kmodules.xyz/custom-resources/client/listers/appcatalog/v1alpha1"
	oc_cs "kmodules.xyz/openshift/client/clientset/versioned"
	oc_informers "kmodules.xyz/openshift/client/informers/externalversions"
	oc_listers "kmodules.xyz/openshift/client/listers/apps/v1"
)

type StashController struct {
	config

	clientConfig     *rest.Config
	kubeClient       kubernetes.Interface
	ocClient         oc_cs.Interface
	stashClient      cs.Interface
	crdClient        crd_cs.ApiextensionsV1beta1Interface
	appCatalogClient appcatalog_cs.Interface
	recorder         record.EventRecorder

	kubeInformerFactory       informers.SharedInformerFactory
	ocInformerFactory         oc_informers.SharedInformerFactory
	stashInformerFactory      stashinformers.SharedInformerFactory
	appCatalogInformerFactory appcatalog_informers.SharedInformerFactory

	// Namespace
	nsInformer cache.SharedIndexInformer

	// Restic
	rstQueue    *queue.Worker
	rstInformer cache.SharedIndexInformer
	rstLister   stash_listers.ResticLister

	// Recovery
	recQueue    *queue.Worker
	recInformer cache.SharedIndexInformer
	recLister   stash_listers.RecoveryLister

	// Repository
	repoQueue    *queue.Worker
	repoInformer cache.SharedIndexInformer
	repoLister   stash_listers.RepositoryLister

	// Deployment
	dpQueue    *queue.Worker
	dpInformer cache.SharedIndexInformer
	dpLister   apps_listers.DeploymentLister

	// DaemonSet
	dsQueue    *queue.Worker
	dsInformer cache.SharedIndexInformer
	dsLister   apps_listers.DaemonSetLister

	// StatefulSet
	ssQueue    *queue.Worker
	ssInformer cache.SharedIndexInformer
	ssLister   apps_listers.StatefulSetLister

	// ReplicationController
	rcQueue    *queue.Worker
	rcInformer cache.SharedIndexInformer
	rcLister   core_listers.ReplicationControllerLister

	// ReplicaSet
	rsQueue    *queue.Worker
	rsInformer cache.SharedIndexInformer
	rsLister   apps_listers.ReplicaSetLister

	// PersistentVolumeClaim
	pvcQueue    *queue.Worker
	pvcInformer cache.SharedIndexInformer
	pvcLister   core_listers.PersistentVolumeClaimLister

	// AppBinding
	abQueue    *queue.Worker
	abInformer cache.SharedIndexInformer
	abLister   abListers.AppBindingLister

	// Job
	jobQueue    *queue.Worker
	jobInformer cache.SharedIndexInformer
	jobLister   batch_listers.JobLister

	// BackupConfiguration
	bcQueue    *queue.Worker
	bcInformer cache.SharedIndexInformer
	bcLister   stash_listers_v1beta1.BackupConfigurationLister
	// BackupSession
	backupSessionQueue    *queue.Worker
	backupSessionInformer cache.SharedIndexInformer
	backupSessionLister   stash_listers_v1beta1.BackupSessionLister

	// RestoreSession
	restoreSessionQueue    *queue.Worker
	restoreSessionInformer cache.SharedIndexInformer
	restoreSessionLister   stash_listers_v1beta1.RestoreSessionLister

	// Openshift DeploymentConfiguration
	dcQueue    *queue.Worker
	dcInformer cache.SharedIndexInformer
	dcLister   oc_listers.DeploymentConfigLister
}

func (c *StashController) ensureCustomResourceDefinitions() error {
	crds := []*crd_api.CustomResourceDefinition{
		api.Restic{}.CustomResourceDefinition(),
		api.Recovery{}.CustomResourceDefinition(),
		api.Repository{}.CustomResourceDefinition(),
		api_v1beta1.BackupConfiguration{}.CustomResourceDefinition(),
		api_v1beta1.BackupSession{}.CustomResourceDefinition(),
		api_v1beta1.Task{}.CustomResourceDefinition(),
		api_v1beta1.Function{}.CustomResourceDefinition(),
		api_v1beta1.BackupConfigurationTemplate{}.CustomResourceDefinition(),
		api_v1beta1.BackupConfiguration{}.CustomResourceDefinition(),
		api_v1beta1.BackupSession{}.CustomResourceDefinition(),
		api_v1beta1.RestoreSession{}.CustomResourceDefinition(),

		appCatalog.AppBinding{}.CustomResourceDefinition(),
	}
	return crdutils.RegisterCRDs(c.crdClient, crds)
}

func (c *StashController) Run(stopCh <-chan struct{}) {
	go c.RunInformers(stopCh)

	if c.EnableMutatingWebhook {
		cancel1, _ := reg_util.SyncMutatingWebhookCABundle(c.clientConfig, mutatingWebhook)
		defer cancel1()
	}
	if c.EnableValidatingWebhook {
		cancel2, _ := reg_util.SyncValidatingWebhookCABundle(c.clientConfig, validatingWebhook)
		defer cancel2()
	}

	<-stopCh
}

func (c *StashController) RunInformers(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	glog.Info("Starting Stash controller")

	c.kubeInformerFactory.Start(stopCh)
	c.stashInformerFactory.Start(stopCh)
	c.appCatalogInformerFactory.Start(stopCh)

	// start ocInformerFactory only if the cluster has DeploymentConfig (for openshift)
	if c.dcInformer != nil {
		c.ocInformerFactory.Start(stopCh)
	}

	// Wait for all involved caches to be synced, before processing items from the queue is started
	for _, v := range c.kubeInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return
		}
	}

	if c.dcInformer != nil {
		for _, v := range c.ocInformerFactory.WaitForCacheSync(stopCh) {
			if !v {
				runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
				return
			}
		}
	}

	for _, v := range c.stashInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return
		}
	}

	for _, v := range c.appCatalogInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return
		}
	}

	// start workload queue
	c.dpQueue.Run(stopCh)
	c.dsQueue.Run(stopCh)
	c.ssQueue.Run(stopCh)
	c.rcQueue.Run(stopCh)
	c.rsQueue.Run(stopCh)

	// start DeploymentConfig queue only if the cluster has DeploymentConfiguration resource (for openshift)
	if c.dcInformer != nil {
		c.dcQueue.Run(stopCh)
	}

	c.pvcQueue.Run(stopCh)
	c.abQueue.Run(stopCh)
	c.jobQueue.Run(stopCh)

	// start v1alpha1 resources queue
	c.repoQueue.Run(stopCh)
	c.rstQueue.Run(stopCh)
	c.recQueue.Run(stopCh)

	// start v1beta1 resources queue
	c.bcQueue.Run(stopCh)
	c.backupSessionQueue.Run(stopCh)
	c.restoreSessionQueue.Run(stopCh)

	<-stopCh
	log.Infoln("Stopping Stash controller")
}
