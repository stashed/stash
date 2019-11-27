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

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/rest"
)

type StandardStorage struct {
	cfg ResourceInfo
}

var _ rest.GroupVersionKindProvider = &StandardStorage{}
var _ rest.Scoper = &StandardStorage{}
var _ rest.StandardStorage = &StandardStorage{}

func NewStandardStorage(cfg ResourceInfo) *StandardStorage {
	return &StandardStorage{cfg}
}

func (r *StandardStorage) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return r.cfg.gvk
}

func (r *StandardStorage) NamespaceScoped() bool {
	return r.cfg.namespaceScoped
}

// Getter
func (r *StandardStorage) New() runtime.Object {
	return r.cfg.obj
}

func (r *StandardStorage) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	return r.New(), nil
}

func (r *StandardStorage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.New(), nil
}

// Lister
func (r *StandardStorage) NewList() runtime.Object {
	return r.cfg.list
}

func (r *StandardStorage) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	return r.NewList(), nil
}

// CreaterUpdater
func (r *StandardStorage) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return r.New(), true, nil
}

// GracefulDeleter
func (r *StandardStorage) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	return r.New(), true, nil
}

// CollectionDeleter
func (r *StandardStorage) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
	return r.NewList(), nil
}

// Watcher
func (r *StandardStorage) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	return nil, nil
}
