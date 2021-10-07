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

package api

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	plugins = map[schema.GroupVersionKind]ResourceCalculator{}
	lock    sync.RWMutex
)

func Register(gvk schema.GroupVersionKind, c ResourceCalculator) {
	lock.Lock()
	plugins[gvk] = c
	lock.Unlock()
}

func Load(obj map[string]interface{}) (ResourceCalculator, error) {
	u := unstructured.Unstructured{Object: obj}
	gvk := u.GroupVersionKind()

	lock.RLock()
	c, ok := plugins[gvk]
	lock.RUnlock()
	if !ok {
		return nil, NotRegistered{gvk}
	}
	return c, nil
}

func RegisteredTypes() []schema.GroupVersionKind {
	lock.RLock()
	result := make([]schema.GroupVersionKind, 0, len(plugins))
	for gvk := range plugins {
		result = append(result, gvk)
	}
	lock.RUnlock()
	return result
}

type NotRegistered struct {
	gvk schema.GroupVersionKind
}

var _ error = NotRegistered{}

func (e NotRegistered) Error() string {
	return fmt.Sprintf("no calculator registered for %v", e.gvk)
}
