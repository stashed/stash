package controller

import (
	"fmt"
	"net/http"

	"github.com/appscode/go/log"
	apiext_util "github.com/appscode/kutil/apiextensions/v1beta1"
	"github.com/appscode/kutil/tools/queue"
	"github.com/appscode/pat"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	cs "github.com/appscode/stash/client/clientset/versioned"
	stashinformers "github.com/appscode/stash/client/informers/externalversions"
	stash_listers "github.com/appscode/stash/client/listers/stash/v1alpha1"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	crd_api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
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
	Config

	KubeClient  kubernetes.Interface
	stashClient cs.Interface
	crdClient   crd_cs.ApiextensionsV1beta1Interface
	recorder    record.EventRecorder

	kubeInformerFactory  informers.SharedInformerFactory
	stashInformerFactory stashinformers.SharedInformerFactory

	// Namespace
	nsInformer cache.SharedIndexInformer

	// Restic
	rstQueue    *queue.Worker
	rstInformer cache.SharedIndexInformer
	RstLister   stash_listers.ResticLister

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

func (c *StashController) ensureCustomResourceDefinitions() error {
	crds := []*crd_api.CustomResourceDefinition{
		api.Restic{}.CustomResourceDefinition(),
		api.Recovery{}.CustomResourceDefinition(),
	}
	return apiext_util.RegisterCRDs(c.crdClient, crds)
}

func (c *StashController) RunInformers(stopCh <-chan struct{}) {
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
	c.jobQueue.Run(stopCh)

	<-stopCh
	glog.Info("Stopping Stash controller")
}

func (c *StashController) Run(stopCh <-chan struct{}) error {
	go c.RunInformers(stopCh)

	m := pat.New()
	m.Get("/metrics", promhttp.Handler())
	http.Handle("/", m)
	log.Infoln("Listening on", c.OpsAddress)
	return http.ListenAndServe(c.OpsAddress, nil)
}
