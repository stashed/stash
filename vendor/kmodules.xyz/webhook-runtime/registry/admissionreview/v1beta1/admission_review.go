/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"context"

	admission "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
)

// Adapted from https://github.com/openshift/generic-admission-server/blob/master/pkg/registry/admissionreview/admission_review.go

type AdmissionHookFunc func(admissionSpec *admission.AdmissionRequest) *admission.AdmissionResponse

type REST struct {
	hookFn AdmissionHookFunc
}

var _ rest.Creater = &REST{}
var _ rest.Scoper = &REST{}
var _ rest.GroupVersionKindProvider = &REST{}

func NewREST(hookFn AdmissionHookFunc) *REST {
	return &REST{
		hookFn: hookFn,
	}
}

func (r *REST) New() runtime.Object {
	return &admission.AdmissionReview{}
}

func (r *REST) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	return admission.SchemeGroupVersion.WithKind("AdmissionReview")
}

func (r *REST) NamespaceScoped() bool {
	return false
}

func (r *REST) Create(ctx context.Context, obj runtime.Object, _ rest.ValidateObjectFunc, _ *metav1.CreateOptions) (runtime.Object, error) {
	admissionReview := obj.(*admission.AdmissionReview)
	admissionReview.Response = r.hookFn(admissionReview.Request)
	return admissionReview, nil
}
