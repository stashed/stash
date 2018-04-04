package snapshot

import (
	"fmt"

	api "github.com/appscode/stash/apis/repositories/v1alpha1"
	"github.com/appscode/stash/client/clientset/versioned"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	restconfig "k8s.io/client-go/rest"
)

type REST struct {
	client versioned.Interface
}

var _ rest.Getter = &REST{}
var _ rest.Lister = &REST{}
var _ rest.Deleter = &REST{}
var _ rest.GroupVersionKindProvider = &REST{}

func NewREST(config *restconfig.Config) *REST {
	return &REST{
		client: versioned.NewForConfigOrDie(config),
	}
}

func (r *REST) New() runtime.Object {
	return &api.Snapshot{}
}

func (r *REST) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return api.SchemeGroupVersion.WithKind(api.ResourceKindSnapshot)
}

func (r *REST) Get(ctx apirequest.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	fmt.Println(">>>>--- GET")
	return r.New(), nil
}

func (r *REST) NewList() runtime.Object {
	return &api.SnapshotList{}
}

func (r *REST) List(ctx apirequest.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	fmt.Println(">>>>--- LIST")
	return r.NewList(), nil
}

func (r *REST) Delete(ctx apirequest.Context, name string) (runtime.Object, error) {
	fmt.Println(">>>>--- DELETE")
	return nil, nil
}
