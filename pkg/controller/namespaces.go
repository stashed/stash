package controller

import (
	"time"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	core_informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func (c *StashController) initNamespaceWatcher() {
	c.nsInformer = c.kubeInformerFactory.InformerFor(&core.Namespace{}, func(client kubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
		return core_informers.NewFilteredNamespaceInformer(
			client,
			resyncPeriod,
			cache.Indexers{},
			nil,
		)
	})
	c.nsInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			if ns, ok := obj.(*core.Namespace); ok {
				items, err := c.rstLister.Restics(ns.Name).List(labels.Everything())
				if err == nil {
					for _, item := range items {
						c.stashClient.StashV1alpha1().Restics(item.Namespace).Delete(item.Name, &metav1.DeleteOptions{})
					}
				}
			}
		},
	})
}
