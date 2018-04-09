package openapi

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime"
)

type ResourceInfo struct {
	gvk  schema.GroupVersionKind
	obj  runtime.Object
	list runtime.Object
}
