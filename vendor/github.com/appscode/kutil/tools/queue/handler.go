package queue

import (
	meta_util "github.com/appscode/kutil/meta"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// QueueingEventHandler queues the key for the object on add and update events
type QueueingEventHandler struct {
	queue         workqueue.RateLimitingInterface
	enqueueAdd    func(obj interface{}) bool
	enqueueUpdate func(oldObj, newObj interface{}) bool
	enqueueDelete bool
}

var _ cache.ResourceEventHandler = &QueueingEventHandler{}

func DefaultEventHandler(queue workqueue.RateLimitingInterface) *QueueingEventHandler {
	return &QueueingEventHandler{
		queue:         queue,
		enqueueAdd:    nil,
		enqueueUpdate: nil,
		enqueueDelete: true,
	}
}

func NewEventHandler(queue workqueue.RateLimitingInterface, enqueueUpdate func(oldObj, newObj interface{}) bool) *QueueingEventHandler {
	return &QueueingEventHandler{
		queue:         queue,
		enqueueAdd:    nil,
		enqueueUpdate: enqueueUpdate,
		enqueueDelete: true,
	}
}

func NewUpsertHandler(queue workqueue.RateLimitingInterface) *QueueingEventHandler {
	return &QueueingEventHandler{
		queue:         queue,
		enqueueAdd:    nil,
		enqueueUpdate: nil,
		enqueueDelete: false,
	}
}

func NewDeleteHandler(queue workqueue.RateLimitingInterface) *QueueingEventHandler {
	return &QueueingEventHandler{
		queue:         queue,
		enqueueAdd:    func(_ interface{}) bool { return false },
		enqueueUpdate: func(_, _ interface{}) bool { return false },
		enqueueDelete: true,
	}
}

func NewObservableHandler(queue workqueue.RateLimitingInterface, enableStatusSubresource bool) *QueueingEventHandler {
	return &QueueingEventHandler{
		queue: queue,
		enqueueAdd: func(o interface{}) bool {
			return !meta_util.AlreadyObserved(o, enableStatusSubresource)
		},
		enqueueUpdate: func(old, nu interface{}) bool {
			return (nu.(metav1.Object)).GetDeletionTimestamp() != nil ||
				!meta_util.AlreadyObserved2(old, nu, enableStatusSubresource)
		},
		enqueueDelete: true,
	}
}

func NewObservableUpdateHandler(queue workqueue.RateLimitingInterface, enableStatusSubresource bool) *QueueingEventHandler {
	return &QueueingEventHandler{
		queue:      queue,
		enqueueAdd: nil,
		enqueueUpdate: func(old, nu interface{}) bool {
			return (nu.(metav1.Object)).GetDeletionTimestamp() != nil ||
				!meta_util.AlreadyObserved2(old, nu, enableStatusSubresource)
		},
		enqueueDelete: true,
	}
}

func Enqueue(queue workqueue.RateLimitingInterface, obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		glog.Errorf("Couldn't get key for object %+v: %v", obj, err)
		return
	}
	queue.Add(key)
}

func (h *QueueingEventHandler) OnAdd(obj interface{}) {
	glog.V(6).Infof("Add event for %+v\n", obj)
	if h.enqueueAdd == nil || h.enqueueAdd(obj) {
		Enqueue(h.queue, obj)
	}
}

func (h *QueueingEventHandler) OnUpdate(oldObj, newObj interface{}) {
	glog.V(6).Infof("Update event for %+v\n", newObj)
	if h.enqueueUpdate == nil || h.enqueueUpdate(oldObj, newObj) {
		Enqueue(h.queue, newObj)
	}
}

func (h *QueueingEventHandler) OnDelete(obj interface{}) {
	glog.V(6).Infof("Delete event for %+v\n", obj)
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
