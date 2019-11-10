/*
Copyright The Kmodules Authors.

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

package openapi

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
)

type GetterStorage struct {
	cfg ResourceInfo
}

var _ rest.GroupVersionKindProvider = &GetterStorage{}
var _ rest.Scoper = &GetterStorage{}
var _ rest.Getter = &GetterStorage{}

func NewGetterStorage(cfg ResourceInfo) *GetterStorage {
	return &GetterStorage{cfg}
}

func (r *GetterStorage) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return r.cfg.gvk
}

func (r *GetterStorage) NamespaceScoped() bool {
	return r.cfg.namespaceScoped
}

// Getter
func (r *GetterStorage) New() runtime.Object {
	return r.cfg.obj
}

func (r *GetterStorage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.New(), nil
}
