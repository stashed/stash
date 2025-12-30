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

	api "go.bytebuilders.dev/audit/api/v1"

	"gomodules.xyz/counter/hourly"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var siteEventCounter hourly.Int32

type SiteInfoPublisher struct {
	p           *EventPublisher
	createEvent EventCreator
}

var _ cache.ResourceEventHandler = &SiteInfoPublisher{}

func (p *SiteInfoPublisher) OnAdd(o any, isInInitialList bool) {
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

	siteEventCounter.Inc()
}

func (p *SiteInfoPublisher) OnUpdate(oldObj, newObj any) {
	uOld, ok := oldObj.(client.Object)
	if !ok {
		return
	}
	uNew, ok := newObj.(client.Object)
	if !ok {
		return
	}

	alreadySentHourly := siteEventCounter.Get() > 0
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

	siteEventCounter.Inc()
}

func (p *SiteInfoPublisher) OnDelete(obj any) {
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

	siteEventCounter.Inc()
}
