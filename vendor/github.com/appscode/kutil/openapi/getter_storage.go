package openapi

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

type GetterStorage struct {
	cfg ResourceInfo
}

var _ rest.GroupVersionKindProvider = &GetterStorage{}
var _ rest.Getter = &GetterStorage{}

func NewGetterStorage(cfg ResourceInfo) *GetterStorage {
	return &GetterStorage{cfg}
}

func (r *GetterStorage) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return r.cfg.gvk
}

// Getter
func (r *GetterStorage) New() runtime.Object {
	return r.cfg.obj
}

func (r *GetterStorage) Get(ctx apirequest.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.New(), nil
}
