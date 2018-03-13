package api

import (
	admission "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	// https://github.com/kubernetes/kubernetes/pull/58025
	_ "k8s.io/kubernetes/pkg/apis/apps/install"
	_ "k8s.io/kubernetes/pkg/apis/batch/install"
	_ "k8s.io/kubernetes/pkg/apis/core/install"
	_ "k8s.io/kubernetes/pkg/apis/extensions/install"
)

type AdmissionHook interface {
	// Initialize is called as a post-start hook
	Initialize(config *rest.Config, stopCh <-chan struct{}) error

	// Resource is the resource to use for hosting your admission webhook. If the hook implements
	// MutatingAdmissionHook as well, the two resources for validating and mutating admission must be different.
	// Note: this is (usually) not the same as the payload resource!
	Resource() (plural schema.GroupVersionResource, singular string)

	// Validate is called to decide whether to accept the admission request. The returned AdmissionResponse may
	// use the Patch field to mutate the object from the passed AdmissionRequest.
	Admit(admissionSpec *admission.AdmissionRequest) *admission.AdmissionResponse
}
