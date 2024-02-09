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

package lib

import (
	"fmt"
	gosync "sync"

	api "go.bytebuilders.dev/audit/api/v1"

	"gomodules.xyz/counter/hourly"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	kmapi "kmodules.xyz/client-go/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceEventPublisher struct {
	p           *EventPublisher
	createEvent EventCreator

	counters map[kmapi.OID]int32
	mu       gosync.Mutex
}

var _ cache.ResourceEventHandler = &ResourceEventPublisher{}

func (p *ResourceEventPublisher) OnAdd(o interface{}, isInInitialList bool) {
	obj, ok := o.(client.Object)
	if !ok {
		return
	}

	ev, err := p.createEvent(obj)
	if err != nil {
		klog.V(5).InfoS("failed to create event data", "error", err)
		return
	}

	if err = p.p.Publish(ev, api.EventCreated); err != nil {
		klog.V(5).InfoS("error while publishing event", "error", err)
		return
	}

	p.mu.Lock()
	oid := kmapi.NewObjectID(obj).OID()
	c := p.counters[oid]
	hourly.Inc32(&c)
	p.counters[oid] = c
	p.mu.Unlock()
}

func (p *ResourceEventPublisher) OnUpdate(oldObj, newObj interface{}) {
	uOld, ok := oldObj.(client.Object)
	if !ok {
		return
	}
	uNew, ok := newObj.(client.Object)
	if !ok {
		return
	}

	alreadySentHourly := false
	oid := kmapi.NewObjectID(uNew).OID()
	p.mu.Lock()
	c := p.counters[oid]
	alreadySentHourly = hourly.Get32(&c) > 0
	p.mu.Unlock()

	if alreadySentHourly &&
		uOld.GetUID() == uNew.GetUID() && uOld.GetGeneration() == uNew.GetGeneration() {
		if klog.V(8).Enabled() {
			klog.V(8).InfoS("skipping update event",
				"gvk", uNew.GetObjectKind().GroupVersionKind(),
				"namespace", uNew.GetNamespace(),
				"name", uNew.GetName(),
			)
		}
		return
	}

	ev, err := p.createEvent(uNew)
	if err != nil {
		klog.V(5).InfoS("failed to create event data", "error", err)
		return
	}

	if err = p.p.Publish(ev, api.EventUpdated); err != nil {
		klog.V(5).InfoS("failed to publish event", "error", err)
		return
	}

	p.mu.Lock()
	c = p.counters[oid]
	hourly.Inc32(&c)
	p.counters[oid] = c
	p.mu.Unlock()
}

func (p *ResourceEventPublisher) OnDelete(obj interface{}) {
	var object client.Object
	var ok bool
	if object, ok = obj.(client.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			klog.V(5).Info("error decoding object, invalid type")
			return
		}
		object, ok = tombstone.Obj.(client.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(5).Infof("Recovered deleted object '%v' from tombstone", tombstone.Obj.(metav1.Object).GetName())
	}

	ev, err := p.createEvent(object)
	if err != nil {
		klog.V(5).InfoS("failed to create event data", "error", err)
		return
	}

	if err := p.p.Publish(ev, api.EventDeleted); err != nil {
		klog.V(5).InfoS("failed to publish event", "error", err)
		return
	}

	p.mu.Lock()
	oid := kmapi.NewObjectID(object).OID()
	c := p.counters[oid]
	hourly.Inc32(&c)
	p.counters[oid] = c
	p.mu.Unlock()
}
