package apiextensions

import (
	crdv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	crdv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

type CustomResourceDefinition struct {
	V1beta1 *crdv1beta1.CustomResourceDefinition
	V1      *crdv1.CustomResourceDefinition
}
