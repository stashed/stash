package api

import (
	"net/http"

	admission "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func StatusUninitialized() *admission.AdmissionResponse {
	status := &admission.AdmissionResponse{}
	status.Allowed = false
	status.Result = &metav1.Status{
		Status: metav1.StatusFailure, Code: http.StatusInternalServerError, Reason: metav1.StatusReasonInternalError,
		Message: "not initialized",
	}
	return status
}

func StatusInternalServerError(err error) *admission.AdmissionResponse {
	status := &admission.AdmissionResponse{}
	status.Allowed = false
	status.Result = &metav1.Status{
		Status: metav1.StatusFailure, Code: http.StatusInternalServerError, Reason: metav1.StatusReasonInternalError,
		Message: err.Error(),
	}
	return status
}

func StatusBadRequest(err error) *admission.AdmissionResponse {
	status := &admission.AdmissionResponse{}
	status.Allowed = false
	status.Result = &metav1.Status{
		Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
		Message: err.Error(),
	}
	return status
}

func StatusForbidden(err error) *admission.AdmissionResponse {
	status := &admission.AdmissionResponse{}
	status.Allowed = false
	status.Result = &metav1.Status{
		Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
		Message: err.Error(),
	}
	return status
}
