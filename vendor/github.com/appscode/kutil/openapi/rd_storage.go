package openapi

import (
	"context"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
)

type RDStorage struct {
	cfg ResourceInfo
}

var _ rest.GroupVersionKindProvider = &RDStorage{}
var _ rest.Scoper = &RDStorage{}
var _ rest.Lister = &RDStorage{}
var _ rest.Getter = &RDStorage{}
var _ rest.GracefulDeleter = &RDStorage{}

func NewRDStorage(cfg ResourceInfo) *RDStorage {
	return &RDStorage{cfg}
}

func (r *RDStorage) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return r.cfg.gvk
}

func (r *RDStorage) NamespaceScoped() bool {
	return r.cfg.namespaceScoped
}

// Getter
func (r *RDStorage) New() runtime.Object {
	return r.cfg.obj
}

func (r *RDStorage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.New(), nil
}

// Lister
func (r *RDStorage) NewList() runtime.Object {
	return r.cfg.list
}

func (r *RDStorage) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	return r.NewList(), nil
}

// Deleter
func (r *RDStorage) Delete(ctx context.Context, name string, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	return r.New(), true, nil
}
