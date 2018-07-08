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
