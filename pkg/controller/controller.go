package controller

import (
	"fmt"

	apiext_util "github.com/appscode/kutil/apiextensions/v1beta1"
	"github.com/appscode/kutil/tools/queue"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	cs "github.com/appscode/stash/client"
	stashinformers "github.com/appscode/stash/informers/externalversions"
	stash_listers "github.com/appscode/stash/listers/stash/v1alpha1"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	crd_api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	apps_listers "k8s.io/client-go/listers/apps/v1beta1"
	batch_listers "k8s.io/client-go/listers/batch/v1"
	core_listers "k8s.io/client-go/listers/core/v1"
	ext_listers "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

type StashController struct {
	k8sClient   kubernetes.Interface
	stashClient cs.Interface
	crdClient   crd_cs.ApiextensionsV1beta1Interface
	options     Options
	recorder    record.EventRecorder

	kubeInformerFactory  informers.SharedInformerFactory
	stashInformerFactory stashinformers.SharedInformerFactory

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

	// Deployment
	dpQueue    *queue.Worker
	dpInformer cache.SharedIndexInformer
	dpLister   apps_listers.DeploymentLister

	// DaemonSet
	dsQueue    *queue.Worker
	dsInformer cache.SharedIndexInformer
	dsLister   ext_listers.DaemonSetLister

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
	rsLister   ext_listers.ReplicaSetLister

	// Job
	jobQueue    *queue.Worker
	jobInformer cache.SharedIndexInformer
	jobLister   batch_listers.JobLister
}

func New(kubeClient kubernetes.Interface, crdClient crd_cs.ApiextensionsV1beta1Interface, stashClient cs.Interface, options Options) *StashController {
	tweakListOptions := func(opt *metav1.ListOptions) {
		opt.IncludeUninitialized = true
	}
	return &StashController{
		k8sClient:            kubeClient,
		stashClient:          stashClient,
		crdClient:            crdClient,
		kubeInformerFactory:  informers.NewFilteredSharedInformerFactory(kubeClient, options.ResyncPeriod, core.NamespaceAll, tweakListOptions),
		stashInformerFactory: stashinformers.NewSharedInformerFactory(stashClient, options.ResyncPeriod),
		options:              options,
		recorder:             eventer.NewEventRecorder(kubeClient, "stash-controller"),
	}
}

func (c *StashController) Setup() error {
	if err := c.ensureCustomResourceDefinitions(); err != nil {
		return err
	}
	if c.options.EnableRBAC {
		if err := c.ensureSidecarClusterRole(); err != nil {
			return err
		}
	}

	c.initNamespaceWatcher()
	c.initResticWatcher()
	c.initRecoveryWatcher()
	c.initDeploymentWatcher()
	c.initDaemonSetWatcher()
	c.initStatefulSetWatcher()
	c.initRCWatcher()
	c.initReplicaSetWatcher()
	c.initJobWatcher()
	return nil
}

func (c *StashController) ensureCustomResourceDefinitions() error {
	crds := []*crd_api.CustomResourceDefinition{
		api.Restic{}.CustomResourceDefinition(),
		api.Recovery{}.CustomResourceDefinition(),
	}
	return apiext_util.RegisterCRDs(c.crdClient, crds)
}

func (c *StashController) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	glog.Info("Starting Stash controller")
	c.kubeInformerFactory.Start(stopCh)
	c.stashInformerFactory.Start(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	for _, v := range c.kubeInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return
		}
	}
	for _, v := range c.stashInformerFactory.WaitForCacheSync(stopCh) {
		if !v {
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
			return
		}
	}

	c.rstQueue.Run(stopCh)
	c.recQueue.Run(stopCh)
	c.dpQueue.Run(stopCh)
	c.dsQueue.Run(stopCh)
	c.ssQueue.Run(stopCh)
	c.rcQueue.Run(stopCh)
	c.rsQueue.Run(stopCh)

	<-stopCh
	glog.Info("Stopping Stash controller")
}
