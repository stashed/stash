package queue

import (
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// QueueingEventHandler queues the key for the object on add and update events
type QueueingEventHandler struct {
	queue         workqueue.RateLimitingInterface
	enqueueAdd    bool
	enqueueUpdate func(oldObj, newObj interface{}) bool
	enqueueDelete bool
}

var _ cache.ResourceEventHandler = &QueueingEventHandler{}

func DefaultEventHandler(queue workqueue.RateLimitingInterface) *QueueingEventHandler {
	return &QueueingEventHandler{
		queue:         queue,
		enqueueAdd:    true,
		enqueueUpdate: nil,
		enqueueDelete: true,
	}
}

func NewEventHandler(queue workqueue.RateLimitingInterface, enqueueUpdate func(oldObj, newObj interface{}) bool) *QueueingEventHandler {
	return &QueueingEventHandler{
		queue:         queue,
		enqueueAdd:    true,
		enqueueUpdate: enqueueUpdate,
		enqueueDelete: true,
	}
}

func NewUpsertHandler(queue workqueue.RateLimitingInterface) *QueueingEventHandler {
	return &QueueingEventHandler{
		queue:         queue,
		enqueueAdd:    true,
		enqueueUpdate: nil,
		enqueueDelete: false,
	}
}

func NewDeleteHandler(queue workqueue.RateLimitingInterface) *QueueingEventHandler {
	return &QueueingEventHandler{
		queue:         queue,
		enqueueAdd:    false,
		enqueueUpdate: func(oldObj, newObj interface{}) bool { return false },
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
	if h.enqueueAdd {
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

// versionedEventHandler is an adaptor to let you easily specify as many or
// as few of the notification functions as you want while still implementing
// ResourceEventHandler.
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
