package stash

import (
	"time"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/runtime"
)

// Like cache.NewInformer but loads the already loaded data and does
// not generate ADD event notification for those.
func NewInformerPopulated(
	lw cache.ListerWatcher,
	objType runtime.Object,
	resyncPeriod time.Duration,
	h cache.ResourceEventHandler,
) (cache.Store, *cache.Controller) {
	store, controller := cache.NewInformer(lw, objType, resyncPeriod, h)

	// watch false on pre population.
	options := kapi.ListOptions{
		ResourceVersion: "0",
		Watch:           false,
	}
	// This will get the object list and add those to clientState, this will prevent
	// ADDED event generation for all the objects all-ready in kube.
	list, err := lw.List(options)
	if err == nil {
		items, err := meta.ExtractList(list)
		if err == nil {
			for _, i := range items {
				store.Add(i)
			}
		}
	}

	return store, controller
}

func NewIndexerInformerPopulated(
	lw cache.ListerWatcher,
	objType runtime.Object,
	resyncPeriod time.Duration,
	h cache.ResourceEventHandler,
	indexers cache.Indexers,
) (cache.Indexer, *cache.Controller) {
	indexer, controller := cache.NewIndexerInformer(lw, objType, resyncPeriod, h, indexers)

	// watch false on pre population.
	options := kapi.ListOptions{
		ResourceVersion: "0",
		Watch:           false,
	}
	// This will get the object list and add those to clientState, this will prevent
	// ADDED event generation for all the objects all-ready in kube.
	list, err := lw.List(options)
	if err == nil {
		items, err := meta.ExtractList(list)
		if err == nil {
			for _, i := range items {
				indexer.Add(i)
			}
		}
	}

	return indexer, controller
}
