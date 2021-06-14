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

package queue

import (
	"reflect"
	"time"

	meta_util "kmodules.xyz/client-go/meta"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// QueueingEventHandler queues the key for the object on add and update events
type QueueingEventHandler struct {
	queue         workqueue.RateLimitingInterface
	enqueueAdd    func(obj interface{}) bool
	enqueueUpdate func(oldObj, newObj interface{}) bool
	enqueueDelete bool
}

var _ cache.ResourceEventHandler = &QueueingEventHandler{}

func DefaultEventHandler(queue workqueue.RateLimitingInterface) cache.ResourceEventHandler {
	return &QueueingEventHandler{
		queue:         queue,
		enqueueAdd:    nil,
		enqueueUpdate: nil,
		enqueueDelete: true,
	}
}

func NewEventHandler(queue workqueue.RateLimitingInterface, enqueueUpdate func(oldObj, newObj interface{}) bool) cache.ResourceEventHandler {
	return &QueueingEventHandler{
		queue:         queue,
		enqueueAdd:    nil,
		enqueueUpdate: enqueueUpdate,
		enqueueDelete: true,
	}
}

func NewUpsertHandler(queue workqueue.RateLimitingInterface) cache.ResourceEventHandler {
	return &QueueingEventHandler{
		queue:         queue,
		enqueueAdd:    nil,
		enqueueUpdate: nil,
		enqueueDelete: false,
	}
}

func NewDeleteHandler(queue workqueue.RateLimitingInterface) cache.ResourceEventHandler {
	return &QueueingEventHandler{
		queue:         queue,
		enqueueAdd:    func(_ interface{}) bool { return false },
		enqueueUpdate: func(_, _ interface{}) bool { return false },
		enqueueDelete: true,
	}
}

func NewReconcilableHandler(queue workqueue.RateLimitingInterface) cache.ResourceEventHandler {
	return &QueueingEventHandler{
		queue: queue,
		enqueueAdd: func(o interface{}) bool {
			return !meta_util.MustAlreadyReconciled(o)
		},
		enqueueUpdate: func(old, nu interface{}) bool {
			return (nu.(metav1.Object)).GetDeletionTimestamp() != nil || !meta_util.MustAlreadyReconciled(nu)
		},
		enqueueDelete: true,
	}
}

func NewChangeHandler(queue workqueue.RateLimitingInterface) cache.ResourceEventHandler {
	return &QueueingEventHandler{
		queue:      queue,
		enqueueAdd: nil,
		enqueueUpdate: func(old, nu interface{}) bool {
			oldObj := old.(metav1.Object)
			nuObj := nu.(metav1.Object)
			return nuObj.GetDeletionTimestamp() != nil ||
				!meta_util.MustAlreadyReconciled(nu) ||
				!reflect.DeepEqual(oldObj.GetLabels(), nuObj.GetLabels()) ||
				!reflect.DeepEqual(oldObj.GetAnnotations(), nuObj.GetAnnotations()) ||
				!meta_util.StatusConditionAwareEqual(old, nu)
		},
		enqueueDelete: true,
	}
}

func NewSpecStatusChangeHandler(queue workqueue.RateLimitingInterface) cache.ResourceEventHandler {
	return &QueueingEventHandler{
		queue:      queue,
		enqueueAdd: nil,
		enqueueUpdate: func(old, nu interface{}) bool {
			nuObj := nu.(metav1.Object)
			return nuObj.GetDeletionTimestamp() != nil ||
				!meta_util.MustAlreadyReconciled(nu) ||
				!meta_util.StatusConditionAwareEqual(old, nu)
		},
		enqueueDelete: true,
	}
}

func Enqueue(queue workqueue.RateLimitingInterface, obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.Errorf("Couldn't get key for object %+v: %v", obj, err)
		return
	}
	queue.Add(key)
}

func EnqueueAfter(queue workqueue.RateLimitingInterface, obj interface{}, duration time.Duration) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.Errorf("Couldn't get key for object %+v: %v", obj, err)
		return
	}
	queue.AddAfter(key, duration)
}

func (h *QueueingEventHandler) OnAdd(obj interface{}) {
	klog.V(6).Infof("Add event for %+v\n", obj)
	if h.enqueueAdd == nil || h.enqueueAdd(obj) {
		Enqueue(h.queue, obj)
	}
}

func (h *QueueingEventHandler) OnUpdate(oldObj, newObj interface{}) {
	klog.V(6).Infof("Update event for %+v\n", newObj)
	if h.enqueueUpdate == nil || h.enqueueUpdate(oldObj, newObj) {
		Enqueue(h.queue, newObj)
	}
}

func (h *QueueingEventHandler) OnDelete(obj interface{}) {
	klog.V(6).Infof("Delete event for %+v\n", obj)
	if h.enqueueDelete {
		Enqueue(h.queue, obj)
	}
}

func NewVersionedHandler(inner cache.ResourceEventHandler, gvk schema.GroupVersionKind) cache.ResourceEventHandler {
	return versionedEventHandler{inner: inner, gvk: gvk}
}

// versionedEventHandler is an adaptor to let you set GroupVersionKind of objects
// while still implementing ResourceEventHandler.
type versionedEventHandler struct {
	inner cache.ResourceEventHandler
	gvk   schema.GroupVersionKind
}

func (w versionedEventHandler) setGroupVersionKind(obj interface{}) interface{} {
	if r, ok := obj.(runtime.Object); ok {
		r = r.DeepCopyObject()
		r.GetObjectKind().SetGroupVersionKind(w.gvk)
		return r
	}
	return obj
}

func (w versionedEventHandler) OnAdd(obj interface{}) {
	w.inner.OnAdd(w.setGroupVersionKind(obj))
}

func (w versionedEventHandler) OnUpdate(oldObj, newObj interface{}) {
	w.inner.OnUpdate(w.setGroupVersionKind(oldObj), w.setGroupVersionKind(newObj))
}

func (w versionedEventHandler) OnDelete(obj interface{}) {
	w.inner.OnDelete(w.setGroupVersionKind(obj))
}

func NewFilteredHandler(inner cache.ResourceEventHandler, sel labels.Selector) cache.ResourceEventHandler {
	return filteredEventHandler{inner: inner, sel: sel}
}

// filteredEventHandler is an adaptor to let you handle event for objects with
// matching label.
type filteredEventHandler struct {
	inner cache.ResourceEventHandler
	sel   labels.Selector
}

func (w filteredEventHandler) matches(obj interface{}) bool {
	accessor, err := meta.Accessor(obj)
	return err == nil && w.sel.Matches(labels.Set(accessor.GetLabels()))
}

func (w filteredEventHandler) OnAdd(obj interface{}) {
	if w.matches(obj) {
		w.inner.OnAdd(obj)
	}
}

func (w filteredEventHandler) OnUpdate(oldObj, newObj interface{}) {
	if w.matches(oldObj) && w.matches(newObj) {
		w.inner.OnUpdate(oldObj, newObj)
	}
}

func (w filteredEventHandler) OnDelete(obj interface{}) {
	if w.matches(obj) {
		w.inner.OnDelete(obj)
	}
}
