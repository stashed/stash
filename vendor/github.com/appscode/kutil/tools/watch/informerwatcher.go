/*
Copyright 2017 The Kubernetes Authors.

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

package watch

import (
	"sync"
	"sync/atomic"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func newTicketer() *ticketer {
	return &ticketer{
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

type ticketer struct {
	counter uint64

	cond    *sync.Cond
	current uint64
}

func (t *ticketer) GetTicket() uint64 {
	// -1 to start from 0
	return atomic.AddUint64(&t.counter, 1) - 1
}

func (t *ticketer) WaitForTicket(ticket uint64, f func()) {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()
	for ticket != t.current {
		t.cond.Wait()
	}

	f()

	t.current++
	t.cond.Broadcast()
}

// NewIndexerInformerWatcher will create an IndexerInformer and wrap it into watch.Interface
// so you can use it anywhere where you'd have used a regular Watcher returned from Watch method.
func NewIndexerInformerWatcher(lw cache.ListerWatcher, objType runtime.Object) (cache.Indexer, cache.Controller, watch.Interface) {
	ch := make(chan watch.Event)
	w := NewProxyWatcher(ch)
	t := newTicketer()

	indexer, informer := cache.NewIndexerInformer(lw, objType, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			go t.WaitForTicket(t.GetTicket(), func() {
				select {
				case ch <- watch.Event{
					Type:   watch.Added,
					Object: obj.(runtime.Object),
				}:
				case <-w.StopChan():
				}
			})
		},
		UpdateFunc: func(old, new interface{}) {
			go t.WaitForTicket(t.GetTicket(), func() {
				select {
				case ch <- watch.Event{
					Type:   watch.Modified,
					Object: new.(runtime.Object),
				}:
				case <-w.StopChan():
				}
			})
		},
		DeleteFunc: func(obj interface{}) {
			go t.WaitForTicket(t.GetTicket(), func() {
				staleObj, stale := obj.(cache.DeletedFinalStateUnknown)
				if stale {
					// We have no means of passing the additional information down using watch API based on watch.Event
					// but the caller can filter such objects by checking if metadata.deletionTimestamp is set
					obj = staleObj
				}

				select {
				case ch <- watch.Event{
					Type:   watch.Deleted,
					Object: obj.(runtime.Object),
				}:
				case <-w.StopChan():
				}
			})
		},
	}, cache.Indexers{})

	go func() {
		informer.Run(w.StopChan())
	}()

	return indexer, informer, w
}

// ProxyWatcher lets you wrap your channel in watch Interface. Threadsafe.
type ProxyWatcher struct {
	result chan watch.Event
	stopCh chan struct{}

	mutex   sync.Mutex
	stopped bool
}

var _ watch.Interface = &ProxyWatcher{}

// NewProxyWatcher creates new ProxyWatcher by wrapping a channel
func NewProxyWatcher(ch chan watch.Event) *ProxyWatcher {
	return &ProxyWatcher{
		result:  ch,
		stopCh:  make(chan struct{}),
		stopped: false,
	}
}

// Stop implements Interface
func (pw *ProxyWatcher) Stop() {
	pw.mutex.Lock()
	defer pw.mutex.Unlock()
	if !pw.stopped {
		pw.stopped = true
		close(pw.stopCh)
	}
}

// Stopping returns true if Stop() has been called
func (pw *ProxyWatcher) Stopping() bool {
	pw.mutex.Lock()
	defer pw.mutex.Unlock()
	return pw.stopped
}

// ResultChan implements Interface
func (pw *ProxyWatcher) ResultChan() <-chan watch.Event {
	return pw.result
}

// StopChan returns stop channel
func (pw *ProxyWatcher) StopChan() <-chan struct{} {
	return pw.stopCh
}
