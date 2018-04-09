package openapi

import (
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

type ListerStorage struct {
	cfg ResourceInfo
}

var _ rest.GroupVersionKindProvider = &ListerStorage{}
var _ rest.Lister = &ListerStorage{}
var _ rest.Getter = &ListerStorage{}

func NewListerStorage(cfg ResourceInfo) *ListerStorage {
	return &ListerStorage{cfg}
}

func (r *ListerStorage) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return r.cfg.gvk
}

// Getter
func (r *ListerStorage) New() runtime.Object {
	return r.cfg.obj
}

func (r *ListerStorage) Get(ctx apirequest.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.New(), nil
}

// Lister
func (r *ListerStorage) NewList() runtime.Object {
	return r.cfg.list
}

func (r *ListerStorage) List(ctx apirequest.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	return r.NewList(), nil
}
