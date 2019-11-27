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

type CDStorage struct {
	cfg ResourceInfo
}

var _ rest.GroupVersionKindProvider = &CDStorage{}
var _ rest.Scoper = &CDStorage{}
var _ rest.Creater = &CDStorage{}
var _ rest.GracefulDeleter = &CDStorage{}

func NewCDStorage(cfg ResourceInfo) *CDStorage {
	return &CDStorage{cfg}
}

func (r *CDStorage) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return r.cfg.gvk
}

func (r *CDStorage) NamespaceScoped() bool {
	return r.cfg.namespaceScoped
}

// Getter
func (r *CDStorage) New() runtime.Object {
	return r.cfg.obj
}

func (r *CDStorage) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	return r.New(), nil
}

// Deleter
func (r *CDStorage) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	return r.New(), true, nil
}
