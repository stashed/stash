package restic

import (
	"encoding/json"
	"net/http"

	api "github.com/appscode/stash/apis/stash/v1alpha1"
	admission "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

type AdmissionHook struct {
}

func (a *AdmissionHook) ValidatingResource() (plural schema.GroupVersionResource, singular string) {
	return schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1alpha1",
			Resource: "resticreviews",
		},
		"resticreview"
}

func (a *AdmissionHook) Validate(admissionSpec *admission.AdmissionRequest) *admission.AdmissionResponse {
	status := &admission.AdmissionResponse{}

	if admissionSpec.Operation != admission.Create || len(admissionSpec.SubResource) != 0 ||
		(admissionSpec.Resource.Group != "stash.appscode.com" || admissionSpec.Resource.Resource != "restics") {
		status.Allowed = true
		return status
	}

	obj := &api.Restic{}
	err := json.Unmarshal(admissionSpec.Object.Raw, obj)
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

	status.Allowed = true
	return status
}

func (a *AdmissionHook) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	return nil
}
