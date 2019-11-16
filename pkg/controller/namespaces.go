/*
Copyright The Stash Authors.

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

package controller

import (
	"time"

	"github.com/appscode/go/log"
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
						err2 := c.stashClient.StashV1alpha1().Restics(item.Namespace).Delete(item.Name, &metav1.DeleteOptions{})
						if err2 != nil {
							log.Errorln(err2)
						}
					}
				}
				// TODO: delete other resources that may cause namespace stuck in terminating state
			}
		},
	})
}
