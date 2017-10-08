package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

func (c *StashController) initNamespaceWatcher() {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (rt.Object, error) {
			return c.k8sClient.CoreV1().Namespaces().List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return c.k8sClient.CoreV1().Namespaces().Watch(options)
		},
	}

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the pod key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Namespace than the version which was responsible for triggering the update.
	c.nsIndexer, c.nsInformer = cache.NewIndexerInformer(lw, &apiv1.Namespace{}, c.options.ResyncPeriod, cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			if ns, ok := obj.(*apiv1.Namespace); ok {
				restics, err := c.stashClient.Restics(ns.Name).List(metav1.ListOptions{})
				if err == nil {
					for _, restic := range restics.Items {
						c.stashClient.Restics(restic.Namespace).Delete(restic.Name, &metav1.DeleteOptions{})
					}
				}
			}
		},
	}, cache.Indexers{})
}
