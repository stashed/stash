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

package factory

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamiclister"
	"k8s.io/client-go/tools/pager"
)

type directImpl struct {
	dc dynamic.Interface

	lock    sync.RWMutex
	listers map[schema.GroupVersionResource]dynamiclister.Lister
}

var _ Factory = &directImpl{}

func (i *directImpl) ForResource(gvr schema.GroupVersionResource) dynamiclister.Lister {
	l := i.existingForResource(gvr)
	if l != nil {
		return l
	}
	return i.newForResource(gvr)
}

func (i *directImpl) newForResource(gvr schema.GroupVersionResource) dynamiclister.Lister {
	i.lock.Lock()
	defer i.lock.Unlock()

	l := newLister(i.dc, gvr)
	i.listers[gvr] = l
	return l
}

func (i *directImpl) existingForResource(gvr schema.GroupVersionResource) dynamiclister.Lister {
	i.lock.RLock()
	defer i.lock.RUnlock()
	l, ok := i.listers[gvr]
	if !ok {
		return nil
	}
	return l
}

var _ dynamiclister.Lister = &dynamicLister{}
var _ dynamiclister.NamespaceLister = &dynamicNamespaceLister{}

// dynamicLister implements the Lister interface.
type dynamicLister struct {
	dc  dynamic.Interface
	gvr schema.GroupVersionResource
}

// newLister returns a new Lister.
func newLister(dc dynamic.Interface, gvr schema.GroupVersionResource) dynamiclister.Lister {
	return &dynamicLister{dc: dc, gvr: gvr}
}

// List lists all resources in the indexer.
func (l *dynamicLister) List(selector labels.Selector) (ret []*unstructured.Unstructured, err error) {
	fn := func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		return l.dc.Resource(l.gvr).List(ctx, opts)
	}
	opts := metav1.ListOptions{
		LabelSelector: selector.String(),
	}
	err = pager.New(fn).EachListItem(context.TODO(), opts, func(obj runtime.Object) error {
		o, ok := obj.(*unstructured.Unstructured)
		if !ok {
			return fmt.Errorf("expected *unstructured.Unstructured, found %s", reflect.TypeOf(obj))
		}
		ret = append(ret, o)
		return nil
	})
	return ret, err
}

// Get retrieves a resource from the indexer with the given name
func (l *dynamicLister) Get(name string) (*unstructured.Unstructured, error) {
	obj, err := l.dc.Resource(l.gvr).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// Namespace returns an object that can list and get resources from a given namespace.
func (l *dynamicLister) Namespace(namespace string) dynamiclister.NamespaceLister {
	return &dynamicNamespaceLister{dc: l.dc, namespace: namespace, gvr: l.gvr}
}

// dynamicNamespaceLister implements the NamespaceLister interface.
type dynamicNamespaceLister struct {
	dc        dynamic.Interface
	namespace string
	gvr       schema.GroupVersionResource
}

// List lists all resources in the indexer for a given namespace.
func (l *dynamicNamespaceLister) List(selector labels.Selector) (ret []*unstructured.Unstructured, err error) {
	fn := func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		return l.dc.Resource(l.gvr).Namespace(l.namespace).List(ctx, opts)
	}
	opts := metav1.ListOptions{
		LabelSelector: selector.String(),
	}
	err = pager.New(fn).EachListItem(context.TODO(), opts, func(obj runtime.Object) error {
		o, ok := obj.(*unstructured.Unstructured)
		if !ok {
			return fmt.Errorf("expected *unstructured.Unstructured, found %s", reflect.TypeOf(obj))
		}
		ret = append(ret, o)
		return nil
	})
	return ret, err
}

// Get retrieves a resource from the indexer for a given namespace and name.
func (l *dynamicNamespaceLister) Get(name string) (*unstructured.Unstructured, error) {
	obj, err := l.dc.Resource(l.gvr).Namespace(l.namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}
