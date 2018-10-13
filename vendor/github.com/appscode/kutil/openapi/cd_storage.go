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
func (r *CDStorage) Delete(ctx context.Context, name string, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	return r.New(), true, nil
}
