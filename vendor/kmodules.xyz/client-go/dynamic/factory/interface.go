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
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/dynamiclister"
)

type Factory interface {
	ForResource(gvr schema.GroupVersionResource) dynamiclister.Lister
}

func New(dc dynamic.Interface) Factory {
	return &directImpl{
		dc:      dc,
		listers: map[schema.GroupVersionResource]dynamiclister.Lister{},
	}
}

func NewCached(dc dynamic.Interface, defaultResync time.Duration, stopCh <-chan struct{}) Factory {
	return &cachedImpl{
		factory: dynamicinformer.NewDynamicSharedInformerFactory(dc, defaultResync),
		stopCh:  stopCh,
		listers: map[schema.GroupVersionResource]dynamiclister.Lister{},
	}
}

func NewFilteredCached(dc dynamic.Interface, defaultResync time.Duration, namespace string, tweakListOptions dynamicinformer.TweakListOptionsFunc, stopCh <-chan struct{}) Factory {
	return &cachedImpl{
		factory: dynamicinformer.NewFilteredDynamicSharedInformerFactory(dc, defaultResync, namespace, tweakListOptions),
		stopCh:  stopCh,
		listers: map[schema.GroupVersionResource]dynamiclister.Lister{},
	}
}

func NewSharedCached(factory dynamicinformer.DynamicSharedInformerFactory, stopCh <-chan struct{}) Factory {
	return &cachedImpl{
		factory: factory,
		stopCh:  stopCh,
		listers: map[schema.GroupVersionResource]dynamiclister.Lister{},
	}
}
