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
	"time"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// Worker continuously runs a Reconcile function against a message Queue
type Worker struct {
	name        string
	queue       workqueue.RateLimitingInterface
	maxRetries  int
	threadiness int
	reconcile   func(key string) error
}

func New(name string, maxRetries, threadiness int, fn func(key string) error) *Worker {
	q := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), name)
	return &Worker{name, q, maxRetries, threadiness, fn}
}

func (w *Worker) GetQueue() workqueue.RateLimitingInterface {
	return w.queue
}

// Run schedules a routine to continuously process Queue messages
// until shutdown is closed
func (w *Worker) Run(shutdown <-chan struct{}) {
	defer runtime.HandleCrash()

	// Every second, process all messages in the Queue until it is time to shutdown
	for i := 0; i < w.threadiness; i++ {
		go wait.Until(w.processQueue, time.Second, shutdown)
	}

	go func() {
		<-shutdown

		// Stop accepting messages into the Queue
		klog.V(1).Infof("Shutting down %s Queue\n", w.name)
		w.queue.ShutDown()
	}()
}

// ProcessAllMessages tries to process all messages in the Queue
func (w *Worker) processQueue() {
	for w.processNextEntry() {
	}
}

// ProcessMessage tries to process the next message in the Queue, and requeues on an error
func (w *Worker) processNextEntry() bool {
	// Wait until there is a new item in the working queue
	key, quit := w.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two deployments with the same key are never processed in
	// parallel.
	defer w.queue.Done(key)

	// Invoke the method containing the business logic
	err := w.reconcile(key.(string))
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		w.queue.Forget(key)
		return true
	}
	klog.Errorf("Failed to process key %v. Reason: %s", key, err)

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if w.queue.NumRequeues(key) < w.maxRetries {
		klog.Infof("Error syncing key %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		w.queue.AddRateLimited(key)
		return true
	}

	w.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	klog.Infof("Dropping key %q out of the queue: %v", key, err)
	return true
}
