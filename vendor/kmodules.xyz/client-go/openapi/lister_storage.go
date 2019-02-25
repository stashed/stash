package openapi

import (
	"context"

	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
)

type ListerStorage struct {
	cfg ResourceInfo
}

var _ rest.GroupVersionKindProvider = &ListerStorage{}
var _ rest.Scoper = &ListerStorage{}
var _ rest.Lister = &ListerStorage{}
var _ rest.Getter = &ListerStorage{}

func NewListerStorage(cfg ResourceInfo) *ListerStorage {
	return &ListerStorage{cfg}
}

func (r *ListerStorage) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return r.cfg.gvk
}

func (r *ListerStorage) NamespaceScoped() bool {
	return r.cfg.namespaceScoped
}

// Getter
func (r *ListerStorage) New() runtime.Object {
	return r.cfg.obj
}

func (r *ListerStorage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.New(), nil
}

// Lister
func (r *ListerStorage) NewList() runtime.Object {
	return r.cfg.list
}

func (r *ListerStorage) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	return r.NewList(), nil
}
