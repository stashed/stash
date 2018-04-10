package openapi

import (
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

type RDStorage struct {
	cfg ResourceInfo
}

var _ rest.GroupVersionKindProvider = &RDStorage{}
var _ rest.Lister = &RDStorage{}
var _ rest.Getter = &RDStorage{}
var _ rest.Deleter = &RDStorage{}

func NewRDStorage(cfg ResourceInfo) *RDStorage {
	return &RDStorage{cfg}
}

func (r *RDStorage) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return r.cfg.gvk
}

// Getter
func (r *RDStorage) New() runtime.Object {
	return r.cfg.obj
}

func (r *RDStorage) Get(ctx apirequest.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.New(), nil
}

// Lister
func (r *RDStorage) NewList() runtime.Object {
	return r.cfg.list
}

func (r *RDStorage) List(ctx apirequest.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	return r.NewList(), nil
}

// Deleter
func (r *RDStorage) Delete(ctx apirequest.Context, name string) (runtime.Object, error) {
	return r.New(), nil
}
