/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package factory

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/dynamiclister"
)

type cachedImpl struct {
	factory dynamicinformer.DynamicSharedInformerFactory
	stopCh  <-chan struct{}

	lock    sync.RWMutex
	listers map[schema.GroupVersionResource]dynamiclister.Lister
}

var _ Factory = &cachedImpl{}

func (i *cachedImpl) ForResource(gvr schema.GroupVersionResource) dynamiclister.Lister {
	l := i.existingForResource(gvr)
	if l != nil {
		return l
	}
	return i.newForResource(gvr)
}

func (i *cachedImpl) newForResource(gvr schema.GroupVersionResource) dynamiclister.Lister {
	i.lock.Lock()
	defer i.lock.Unlock()

	informerDep := i.factory.ForResource(gvr)
	i.factory.Start(i.stopCh)
	if synced := i.factory.WaitForCacheSync(i.stopCh); !synced[gvr] {
		panic(fmt.Sprintf("informer for %s hasn't synced", gvr))
	}
	l := dynamiclister.New(informerDep.Informer().GetIndexer(), gvr)
	i.listers[gvr] = l
	return l
}

func (i *cachedImpl) existingForResource(gvr schema.GroupVersionResource) dynamiclister.Lister {
	i.lock.RLock()
	defer i.lock.RUnlock()
	l, ok := i.listers[gvr]
	if !ok {
		return nil
	}
	return l
}
