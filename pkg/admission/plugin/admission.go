package plugin

import (
	"encoding/json"
	"net/http"

	api "github.com/appscode/stash/apis/stash/v1alpha1"
	hookapi "github.com/appscode/stash/pkg/admission/api"
	admission "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
)

type CRDValidator struct {
}

var _ hookapi.AdmissionHook = &CRDValidator{}

func (a *CRDValidator) Resource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "admissionreviews",
		},
		"admissionreview"
}

func (a *CRDValidator) Admit(req *admission.AdmissionRequest) *admission.AdmissionResponse {
	status := &admission.AdmissionResponse{}
	supportedKinds := sets.NewString(api.ResourceKindRestic, api.ResourceKindRecovery)

	if (req.Operation != admission.Create && req.Operation != admission.Update) ||
		len(req.SubResource) != 0 ||
		req.Kind.Group != api.SchemeGroupVersion.Group ||
		!supportedKinds.Has(req.Kind.Kind) {
		status.Allowed = true
		return status
	}

	switch req.Kind.Kind {
	case api.ResourceKindRestic:
		obj := &api.Restic{}
		err := json.Unmarshal(req.Object.Raw, obj)
		if err != nil {
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
				Message: err.Error(),
			}
			return status
		}
		err = obj.IsValid()
		if err != nil {
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
				Message: err.Error(),
			}
			return status
		}
	case api.ResourceKindRecovery:
		obj := &api.Recovery{}
		err := json.Unmarshal(req.Object.Raw, obj)
		if err != nil {
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
				Message: err.Error(),
			}
			return status
		}
		err = obj.IsValid()
		if err != nil {
			status.Allowed = false
			status.Result = &metav1.Status{
				Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
				Message: err.Error(),
			}
			return status
		}
	}

	status.Allowed = true
	return status
}

func (a *CRDValidator) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	return nil
}
