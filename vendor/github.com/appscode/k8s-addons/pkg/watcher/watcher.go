package watcher

import (
	"reflect"
	"sync"
	"time"

	acs "github.com/appscode/k8s-addons/client/clientset"
	"github.com/appscode/k8s-addons/pkg/events"
	"github.com/appscode/k8s-addons/pkg/stash"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/runtime"
)

type Watcher struct {
	// kubernetes client to apiserver
	Client clientset.Interface

	// client for getting the appscode extensions
	AppsCodeExtensionClient acs.AppsCodeExtensionInterface

	// sync time to sync the list.
	SyncPeriod time.Duration

	// lister store
	Storage *stash.Storage

	Dispatch func(e *events.Event) error

	sync.Mutex
}

func (k *Watcher) Cache(resource events.ObjectType, object runtime.Object, lw *cache.ListWatch) (cache.Store, *cache.Controller) {
	var listWatch *cache.ListWatch
	if lw != nil {
		listWatch = lw
	} else {
		listWatch = cache.NewListWatchFromClient(k.Client.Core().RESTClient(), resource.String(), kapi.NamespaceAll, fields.Everything())
	}

	return cache.NewInformer(
		listWatch,
		object,
		k.SyncPeriod,
		eventHandlerFuncs(k),
	)
}

func (k *Watcher) CacheStore(resource events.ObjectType, object runtime.Object, lw *cache.ListWatch) (cache.Store, *cache.Controller) {
	if lw == nil {
		lw = cache.NewListWatchFromClient(k.Client.Core().RESTClient(), resource.String(), kapi.NamespaceAll, fields.Everything())
	}

	return stash.NewInformerPopulated(
		lw,
		object,
		k.SyncPeriod,
		eventHandlerFuncs(k),
	)
}

func (k *Watcher) CacheIndexer(resource events.ObjectType, object runtime.Object, lw *cache.ListWatch, indexers cache.Indexers) (cache.Indexer, *cache.Controller) {
	if lw == nil {
		lw = cache.NewListWatchFromClient(k.Client.Core().RESTClient(), resource.String(), kapi.NamespaceAll, fields.Everything())
	}
	if indexers == nil {
		indexers = cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}
	}

	return stash.NewIndexerInformerPopulated(
		lw,
		object,
		k.SyncPeriod,
		eventHandlerFuncs(k),
		indexers,
	)
}

func eventHandlerFuncs(k *Watcher) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			e := events.New(events.Added, obj)
			k.Dispatch(e)
		},
		DeleteFunc: func(obj interface{}) {
			e := events.New(events.Deleted, obj)
			k.Dispatch(e)
		},
		UpdateFunc: func(old, new interface{}) {
			if !reflect.DeepEqual(old, new) {
				e := events.New(events.Updated, old, new)
				k.Dispatch(e)
			}
		},
	}
}
