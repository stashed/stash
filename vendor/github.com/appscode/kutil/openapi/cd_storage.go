package openapi

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

type CDStorage struct {
	cfg ResourceInfo
}

var _ rest.GroupVersionKindProvider = &CDStorage{}
var _ rest.Creater = &CDStorage{}
var _ rest.GracefulDeleter = &CDStorage{}

func NewCDStorage(cfg ResourceInfo) *CDStorage {
	return &CDStorage{cfg}
}

func (r *CDStorage) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return r.cfg.gvk
}

// Getter
func (r *CDStorage) New() runtime.Object {
	return r.cfg.obj
}

func (r *CDStorage) Create(ctx apirequest.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, includeUninitialized bool) (runtime.Object, error) {
	return r.New(), nil
}

// Deleter
func (r *CDStorage) Delete(ctx apirequest.Context, name string, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	return r.New(), true, nil
}
