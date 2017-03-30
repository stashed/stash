package watcher

import (
	aci "github.com/appscode/k8s-addons/api"
	"github.com/appscode/k8s-addons/pkg/events"
	"github.com/appscode/log"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/apps"
	ext "k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/util/wait"
)

func (k *Watcher) Namespace() {
	log.Debugln("watching", events.Namespace.String())
	lw := &cache.ListWatch{
		ListFunc:  NamespaceListFunc(k.Client),
		WatchFunc: NamespaceWatchFunc(k.Client),
	}
	_, controller := k.Cache(events.Namespace, &kapi.Namespace{}, lw)
	go controller.Run(wait.NeverStop)
}

func (k *Watcher) Pod() {
	log.Debugln("watching", events.Pod.String())
	lw := &cache.ListWatch{
		ListFunc:  PodListFunc(k.Client),
		WatchFunc: PodWatchFunc(k.Client),
	}
	indexer, controller := k.CacheIndexer(events.Pod, &kapi.Pod{}, lw, nil)
	go controller.Run(wait.NeverStop)
	k.Storage.PodStore = cache.StoreToPodLister{indexer}
}

func (k *Watcher) Service() {
	log.Debugln("watching", events.Service.String())
	lw := &cache.ListWatch{
		ListFunc:  ServiceListFunc(k.Client),
		WatchFunc: ServiceWatchFunc(k.Client),
	}
	indexer, controller := k.CacheIndexer(events.Service, &kapi.Service{}, lw, nil)
	go controller.Run(wait.NeverStop)
	k.Storage.ServiceStore = cache.StoreToServiceLister{indexer}
}

func (k *Watcher) RC() {
	log.Debugln("watching", events.RC.String())
	lw := &cache.ListWatch{
		ListFunc:  ReplicationControllerListFunc(k.Client),
		WatchFunc: ReplicationControllerWatchFunc(k.Client),
	}
	indexer, controller := k.CacheIndexer(events.RC, &kapi.ReplicationController{}, lw, nil)
	go controller.Run(wait.NeverStop)
	k.Storage.RcStore = cache.StoreToReplicationControllerLister{indexer}
}

func (k *Watcher) ReplicaSet() {
	log.Debugln("watching", events.ReplicaSet.String())
	lw := &cache.ListWatch{
		ListFunc:  ReplicaSetListFunc(k.Client),
		WatchFunc: ReplicaSetWatchFunc(k.Client),
	}
	indexer, controller := k.CacheIndexer(events.ReplicaSet, &ext.ReplicaSet{}, lw, nil)
	go controller.Run(wait.NeverStop)
	k.Storage.ReplicaSetStore = cache.StoreToReplicaSetLister{indexer}
}

func (k *Watcher) StatefulSet() {
	log.Debugln("watching", events.StatefulSet.String())
	lw := &cache.ListWatch{
		ListFunc:  StatefulSetListFunc(k.Client),
		WatchFunc: StatefulSetWatchFunc(k.Client),
	}
	indexer, controller := k.CacheIndexer(events.StatefulSet, &apps.StatefulSet{}, lw, nil)
	go controller.Run(wait.NeverStop)
	k.Storage.StatefulSetStore = cache.StoreToStatefulSetLister{indexer}
}

func (k *Watcher) DaemonSet() {
	log.Debugln("watching", events.DaemonSet.String())
	lw := &cache.ListWatch{
		ListFunc:  DaemonSetListFunc(k.Client),
		WatchFunc: DaemonSetWatchFunc(k.Client),
	}
	indexer, controller := k.CacheIndexer(events.DaemonSet, &ext.DaemonSet{}, lw, nil)
	go controller.Run(wait.NeverStop)
	k.Storage.DaemonSetStore = cache.StoreToDaemonSetLister{indexer}
}

func (k *Watcher) Endpoint() {
	log.Debugln("watching", events.Endpoint.String())
	lw := &cache.ListWatch{
		ListFunc:  EndpointListFunc(k.Client),
		WatchFunc: EndpointWatchFunc(k.Client),
	}
	store, controller := k.CacheStore(events.Endpoint, &kapi.Endpoints{}, lw)
	go controller.Run(wait.NeverStop)
	k.Storage.EndpointStore = cache.StoreToEndpointsLister{store}
}

func (k *Watcher) Node() {
	log.Debugln("watching", events.Node.String())
	lw := &cache.ListWatch{
		ListFunc:  NodeListFunc(k.Client),
		WatchFunc: NodeWatchFunc(k.Client),
	}
	_, controller := k.CacheStore(events.Node, &kapi.Node{}, lw)
	go controller.Run(wait.NeverStop)
}

func (k *Watcher) Ingress() {
	log.Debugln("watching", events.Ingress.String())
	lw := &cache.ListWatch{
		ListFunc:  IngressListFunc(k.Client),
		WatchFunc: IngressWatchFunc(k.Client),
	}
	_, controller := k.Cache(events.Ingress, &ext.Ingress{}, lw)
	go controller.Run(wait.NeverStop)
}

func (k *Watcher) ExtendedIngress() {
	log.Debugln("watching", events.ExtendedIngress.String())
	lw := &cache.ListWatch{
		ListFunc:  ExtendedIngressListFunc(k.AppsCodeExtensionClient),
		WatchFunc: ExtendedIngressWatchFunc(k.AppsCodeExtensionClient),
	}
	_, controller := k.Cache(events.ExtendedIngress, &aci.Ingress{}, lw)
	go controller.Run(wait.NeverStop)
}

func (k *Watcher) Alert() {
	log.Debugln("watching", events.Alert.String())
	lw := &cache.ListWatch{
		ListFunc:  AlertListFunc(k.AppsCodeExtensionClient),
		WatchFunc: AlertWatchFunc(k.AppsCodeExtensionClient),
	}
	_, controller := k.Cache(events.Alert, &aci.Alert{}, lw)
	go controller.Run(wait.NeverStop)
}

func (k *Watcher) Certificate() {
	log.Debugln("watching", events.Certificate.String())
	lw := &cache.ListWatch{
		ListFunc:  CertificateListFunc(k.AppsCodeExtensionClient),
		WatchFunc: CertificateWatchFunc(k.AppsCodeExtensionClient),
	}
	_, controller := k.Cache(events.Certificate, &aci.Certificate{}, lw)
	go controller.Run(wait.NeverStop)
}

func (k *Watcher) AlertEvent() {
	log.Debugln("watching", events.AlertEvent.String())
	lw := &cache.ListWatch{
		ListFunc:  AlertEventListFunc(k.Client),
		WatchFunc: AlertEventWatchFunc(k.Client),
	}
	_, controller := k.Cache(events.AlertEvent, &kapi.Event{}, lw)
	go controller.Run(wait.NeverStop)
}

func (k *Watcher) Deployment() {
	log.Debugln("watching", events.Deployments.String())
	lw := &cache.ListWatch{
		ListFunc:  DeploymentListFunc(k.Client),
		WatchFunc: DeploymentWatchFunc(k.Client),
	}
	indexer, controller := k.CacheIndexer(events.Deployments, &ext.Deployment{}, lw, nil)
	go controller.Run(wait.NeverStop)
	k.Storage.DeploymentStore = cache.StoreToDeploymentLister{indexer}
}
