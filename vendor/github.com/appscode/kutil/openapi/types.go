package openapi

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ResourceInfo struct {
	gvk  schema.GroupVersionKind
	obj  runtime.Object
	list runtime.Object
}
