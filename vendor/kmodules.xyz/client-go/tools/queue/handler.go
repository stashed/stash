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
	"fmt"
	"strings"
	"time"

	meta_util "kmodules.xyz/client-go/meta"

	core "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// NamespaceDemo means the object is in the demo namespace
	NamespaceDemo string = "demo"
)

var (
	appsCodeAPIGroups = sets.NewString(
		"appscode.com",
		"kubedb.com",
		"kubevault.com",
		"kubeform.com",
	)
)

func logLevel(apiGroup string) klog.Level {
	if appsCodeAPIGroups.Has(apiGroup) {
		return 3
	}
	for g := range appsCodeAPIGroups {
		if strings.HasSuffix(apiGroup, "."+g) {
			return 3
		}
	}
	return 8
}

// QueueingEventHandler queues the key for the object on add and update events
type QueueingEventHandler struct {
	queue               workqueue.RateLimitingInterface
	enqueueAdd          func(obj interface{}) bool
	enqueueUpdate       func(oldObj, newObj interface{}) bool
	enqueueDelete       bool
	restrictToNamespace string
}

var _ cache.ResourceEventHandler = &QueueingEventHandler{}

func DefaultEventHandler(queue workqueue.RateLimitingInterface, restrictToNamespace string) cache.ResourceEventHandler {
	return &QueueingEventHandler{
		queue:               queue,
		enqueueAdd:          nil,
		enqueueUpdate:       nil,
		enqueueDelete:       true,
		restrictToNamespace: restrictToNamespace,
	}
}

func NewEventHandler(queue workqueue.RateLimitingInterface, enqueueUpdate func(oldObj, newObj interface{}) bool, restrictToNamespace string) cache.ResourceEventHandler {
	return &QueueingEventHandler{
		queue:               queue,
		enqueueAdd:          nil,
		enqueueUpdate:       enqueueUpdate,
		enqueueDelete:       true,
		restrictToNamespace: restrictToNamespace,
	}
}

func NewUpsertHandler(queue workqueue.RateLimitingInterface, restrictToNamespace string) cache.ResourceEventHandler {
	return &QueueingEventHandler{
		queue:               queue,
		enqueueAdd:          nil,
		enqueueUpdate:       nil,
		enqueueDelete:       false,
		restrictToNamespace: restrictToNamespace,
	}
}

func NewDeleteHandler(queue workqueue.RateLimitingInterface, restrictToNamespace string) cache.ResourceEventHandler {
	return &QueueingEventHandler{
		queue:               queue,
		enqueueAdd:          func(_ interface{}) bool { return false },
		enqueueUpdate:       func(_, _ interface{}) bool { return false },
		enqueueDelete:       true,
		restrictToNamespace: restrictToNamespace,
	}
}

func NewReconcilableHandler(queue workqueue.RateLimitingInterface, restrictToNamespace string) cache.ResourceEventHandler {
	return &QueueingEventHandler{
		queue: queue,
		enqueueAdd: func(o interface{}) bool {
			return !meta_util.MustAlreadyReconciled(o)
		},
		enqueueUpdate: func(old, nu interface{}) bool {
			return (nu.(metav1.Object)).GetDeletionTimestamp() != nil || !meta_util.MustAlreadyReconciled(nu)
		},
		enqueueDelete:       true,
		restrictToNamespace: restrictToNamespace,
	}
}

func NewChangeHandler(queue workqueue.RateLimitingInterface, restrictToNamespace string) cache.ResourceEventHandler {
	return &QueueingEventHandler{
		queue:      queue,
		enqueueAdd: nil,
		enqueueUpdate: func(old, nu interface{}) bool {
			oldObj := old.(metav1.Object)
			nuObj := nu.(metav1.Object)
			return nuObj.GetDeletionTimestamp() != nil ||
				!meta_util.MustAlreadyReconciled(nu) ||
				!apiequality.Semantic.DeepEqual(oldObj.GetLabels(), nuObj.GetLabels()) ||
				!apiequality.Semantic.DeepEqual(oldObj.GetAnnotations(), nuObj.GetAnnotations()) ||
				!meta_util.StatusConditionAwareEqual(old, nu)
		},
		enqueueDelete:       true,
		restrictToNamespace: restrictToNamespace,
	}
}

func NewSpecStatusChangeHandler(queue workqueue.RateLimitingInterface, restrictToNamespace string) cache.ResourceEventHandler {
	return &QueueingEventHandler{
		queue:      queue,
		enqueueAdd: nil,
		enqueueUpdate: func(old, nu interface{}) bool {
			nuObj := nu.(metav1.Object)
			return nuObj.GetDeletionTimestamp() != nil ||
				!meta_util.MustAlreadyReconciled(nu) ||
				!meta_util.StatusConditionAwareEqual(old, nu)
		},
		enqueueDelete:       true,
		restrictToNamespace: restrictToNamespace,
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
		if h.restrictToNamespace != core.NamespaceAll {
			o, ok := obj.(client.Object)
			if !ok {
				return
			}
			if o.GetNamespace() != "" && o.GetNamespace() != h.restrictToNamespace {
				// WARNING: o.GetObjectKind().GroupVersionKind() is not set and can't be used to detect GVK
				if gvks, _, _ := clientsetscheme.Scheme.ObjectKinds(o); len(gvks) > 0 {
					klog.
						V(logLevel(gvks[0].Group)).
						Infof("Skipping %v %s/%s. Only %s namespace is supported for Community Edition. Please upgrade to Enterprise to use any namespace.", gvks[0], o.GetNamespace(), o.GetName(), h.restrictToNamespace)
				}
				return
			}
		}

		Enqueue(h.queue, obj)
	}
}

func (h *QueueingEventHandler) OnUpdate(oldObj, newObj interface{}) {
	klog.V(6).Infof("Update event for %+v\n", newObj)
	if h.enqueueUpdate == nil || h.enqueueUpdate(oldObj, newObj) {
		if h.restrictToNamespace != core.NamespaceAll {
			o, ok := newObj.(client.Object)
			if !ok {
				return
			}
			if o.GetNamespace() != "" && o.GetNamespace() != h.restrictToNamespace {
				// WARNING: o.GetObjectKind().GroupVersionKind() is not set and can't be used to detect GVK
				if gvks, _, _ := clientsetscheme.Scheme.ObjectKinds(o); len(gvks) > 0 {
					klog.
						V(logLevel(gvks[0].Group)).
						Infof("Skipping %v %s/%s. Only %s namespace is supported for Community Edition. Please upgrade to Enterprise to use any namespace.", gvks[0], o.GetNamespace(), o.GetName(), h.restrictToNamespace)
				}
				return
			}
		}

		Enqueue(h.queue, newObj)
	}
}

func (h *QueueingEventHandler) OnDelete(obj interface{}) {
	klog.V(6).Infof("Delete event for %+v\n", obj)
	if h.enqueueDelete {
		if h.restrictToNamespace != core.NamespaceAll {
			var o client.Object
			var ok bool
			if o, ok = obj.(client.Object); !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					klog.V(5).Info("error decoding object, invalid type")
					return
				}
				o, ok = tombstone.Obj.(client.Object)
				if !ok {
					utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
					return
				}
				klog.V(5).Infof("Recovered deleted object '%v' from tombstone", tombstone.Obj.(metav1.Object).GetName())
			}
			if o.GetNamespace() != "" && o.GetNamespace() != h.restrictToNamespace {
				// WARNING: o.GetObjectKind().GroupVersionKind() is not set and can't be used to detect GVK
				if gvks, _, _ := clientsetscheme.Scheme.ObjectKinds(o); len(gvks) > 0 {
					klog.
						V(logLevel(gvks[0].Group)).
						Infof("Skipping %v %s/%s. Only %s namespace is supported for Community Edition. Please upgrade to Enterprise to use any namespace.", gvks[0], o.GetNamespace(), o.GetName(), h.restrictToNamespace)
				}
				return
			}
		}

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
