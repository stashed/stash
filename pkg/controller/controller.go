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

	api "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned"
	stashinformers "stash.appscode.dev/apimachinery/client/informers/externalversions"
	stash_listers "stash.appscode.dev/apimachinery/client/listers/stash/v1alpha1"
	stash_listers_v1beta1 "stash.appscode.dev/apimachinery/client/listers/stash/v1beta1"

	crd_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	apps_listers "k8s.io/client-go/listers/apps/v1"
	batch_listers "k8s.io/client-go/listers/batch/v1"
	core_listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	reg_util "kmodules.xyz/client-go/admissionregistration/v1beta1"
	"kmodules.xyz/client-go/apiextensions"
	"kmodules.xyz/client-go/tools/queue"
	appCatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
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
	crdClient        crd_cs.Interface
	appCatalogClient appcatalog_cs.Interface
	recorder         record.EventRecorder
	auditor          cache.ResourceEventHandler

	kubeInformerFactory  informers.SharedInformerFactory
	ocInformerFactory    oc_informers.SharedInformerFactory
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
	crds := []*apiextensions.CustomResourceDefinition{
		api.Restic{}.CustomResourceDefinition(),
		api.Recovery{}.CustomResourceDefinition(),
		api.Repository{}.CustomResourceDefinition(),
		api_v1beta1.BackupConfiguration{}.CustomResourceDefinition(),
		api_v1beta1.BackupSession{}.CustomResourceDefinition(),
		api_v1beta1.BackupBlueprint{}.CustomResourceDefinition(),
		api_v1beta1.RestoreSession{}.CustomResourceDefinition(),
		api_v1beta1.RestoreBatch{}.CustomResourceDefinition(),
		api_v1beta1.Task{}.CustomResourceDefinition(),
		api_v1beta1.Function{}.CustomResourceDefinition(),

		appCatalog.AppBinding{}.CustomResourceDefinition(),
	}
	return apiextensions.RegisterCRDs(c.crdClient, crds)
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

	klog.Info("Starting Stash controller")

	c.kubeInformerFactory.Start(stopCh)
	c.stashInformerFactory.Start(stopCh)

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
	klog.Infoln("Stopping Stash controller")
}
