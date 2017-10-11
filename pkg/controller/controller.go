package controller

import (
	"fmt"
	"time"

	"github.com/appscode/kutil"
	sapi "github.com/appscode/stash/apis/stash"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	cs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	stash_listers "github.com/appscode/stash/listers/stash/v1alpha1"
	"github.com/appscode/stash/pkg/eventer"
	"github.com/golang/glog"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	apps_listers "k8s.io/client-go/listers/apps/v1beta1"
	core_listers "k8s.io/client-go/listers/core/v1"
	ext_listers "k8s.io/client-go/listers/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

type StashController struct {
	k8sClient   kubernetes.Interface
	stashClient cs.StashV1alpha1Interface
	crdClient   apiextensionsclient.Interface
	options     Options
	recorder    record.EventRecorder

	// Namespace
	nsIndexer  cache.Indexer
	nsInformer cache.Controller

	// Restic
	rQueue    workqueue.RateLimitingInterface
	rIndexer  cache.Indexer
	rInformer cache.Controller
	rLister   stash_listers.ResticLister

	// Recovery
	rcvQueue    workqueue.RateLimitingInterface
	rcvIndexer  cache.Indexer
	rcvInformer cache.Controller
	rcvLister   stash_listers.ResticLister

	// Deployment
	dpQueue    workqueue.RateLimitingInterface
	dpIndexer  cache.Indexer
	dpInformer cache.Controller
	dpLister   apps_listers.DeploymentLister

	// DaemonSet
	dsQueue    workqueue.RateLimitingInterface
	dsIndexer  cache.Indexer
	dsInformer cache.Controller
	dsLister   ext_listers.DaemonSetLister

	// StatefulSet
	ssQueue    workqueue.RateLimitingInterface
	ssIndexer  cache.Indexer
	ssInformer cache.Controller
	ssLister   apps_listers.StatefulSetLister

	// ReplicationController
	rcQueue    workqueue.RateLimitingInterface
	rcIndexer  cache.Indexer
	rcInformer cache.Controller
	rcLister   core_listers.ReplicationControllerLister

	// ReplicaSet
	rsQueue    workqueue.RateLimitingInterface
	rsIndexer  cache.Indexer
	rsInformer cache.Controller
	rsLister   ext_listers.ReplicaSetLister
}

func New(kubeClient kubernetes.Interface, crdClient apiextensionsclient.Interface, stashClient cs.StashV1alpha1Interface, options Options) *StashController {
	return &StashController{
		k8sClient:   kubeClient,
		stashClient: stashClient,
		crdClient:   crdClient,
		options:     options,
		recorder:    eventer.NewEventRecorder(kubeClient, "stash-controller"),
	}
}

func (c *StashController) Setup() error {
	if err := c.ensureCustomResourceDefinitions(); err != nil {
		return err
	}
	c.initNamespaceWatcher()
	c.initResticWatcher()
	c.initDeploymentWatcher()
	c.initDaemonSetWatcher()
	// c.initStatefulSetWatcher()
	c.initRCWatcher()
	c.initReplicaSetWatcher()
	return nil
}

func (c *StashController) ensureCustomResourceDefinitions() error {
	crds := []*apiextensions.CustomResourceDefinition{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   sapi.ResourceTypeRestic + "." + api.SchemeGroupVersion.Group,
				Labels: map[string]string{"app": "stash"},
			},
			Spec: apiextensions.CustomResourceDefinitionSpec{
				Group:   sapi.GroupName,
				Version: api.SchemeGroupVersion.Version,
				Scope:   apiextensions.NamespaceScoped,
				Names: apiextensions.CustomResourceDefinitionNames{
					Singular:   sapi.ResourceNameRestic,
					Plural:     sapi.ResourceTypeRestic,
					Kind:       sapi.ResourceKindRestic,
					ShortNames: []string{"rst"},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:   sapi.ResourceTypeRecovery + "." + api.SchemeGroupVersion.Group,
				Labels: map[string]string{"app": "stash"},
			},
			Spec: apiextensions.CustomResourceDefinitionSpec{
				Group:   sapi.GroupName,
				Version: api.SchemeGroupVersion.Version,
				Scope:   apiextensions.NamespaceScoped,
				Names: apiextensions.CustomResourceDefinitionNames{
					Singular:   sapi.ResourceNameRecovery,
					Plural:     sapi.ResourceTypeRecovery,
					Kind:       sapi.ResourceKindRecovery,
					ShortNames: []string{"rcv"},
				},
			},
		},
	}
	for _, crd := range crds {
		_, err := c.crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			_, err = c.crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
			if err != nil {
				return err
			}
		}
	}
	return kutil.WaitForCRDReady(c.k8sClient.CoreV1().RESTClient(), crds)
}

func (c *StashController) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.rQueue.ShutDown()
	defer c.dpQueue.ShutDown()
	defer c.dsQueue.ShutDown()
	// defer c.ssQueue.ShutDown()
	defer c.rcQueue.ShutDown()
	defer c.rsQueue.ShutDown()
	glog.Info("Starting Stash controller")

	go c.nsInformer.Run(stopCh)
	go c.rInformer.Run(stopCh)
	go c.dpInformer.Run(stopCh)
	go c.dsInformer.Run(stopCh)
	// go c.ssInformer.Run(stopCh)
	go c.rcInformer.Run(stopCh)
	go c.rsInformer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.nsInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if !cache.WaitForCacheSync(stopCh, c.rInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if !cache.WaitForCacheSync(stopCh, c.dsInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if !cache.WaitForCacheSync(stopCh, c.dpInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if !cache.WaitForCacheSync(stopCh, c.rcInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	if !cache.WaitForCacheSync(stopCh, c.rsInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}
	//if !cache.WaitForCacheSync(stopCh, c.ssInformer.HasSynced) {
	//	runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
	//	return
	//}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runResticWatcher, time.Second, stopCh)
		go wait.Until(c.runDeploymentWatcher, time.Second, stopCh)
		go wait.Until(c.runDaemonSetWatcher, time.Second, stopCh)
		// go wait.Until(c.runStatefulSetWatcher, time.Second, stopCh)
		go wait.Until(c.runRCWatcher, time.Second, stopCh)
		go wait.Until(c.runReplicaSetWatcher, time.Second, stopCh)
	}

	<-stopCh
	glog.Info("Stopping Stash controller")
}
