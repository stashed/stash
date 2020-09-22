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
	"net/http"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func StatusUninitialized() *v1beta1.AdmissionResponse {
	status := &v1beta1.AdmissionResponse{}
	status.Allowed = false
	status.Result = &metav1.Status{
		Status: metav1.StatusFailure, Code: http.StatusInternalServerError, Reason: metav1.StatusReasonInternalError,
		Message: "not initialized",
	}
	return status
}

func StatusInternalServerError(err error) *v1beta1.AdmissionResponse {
	status := &v1beta1.AdmissionResponse{}
	status.Allowed = false
	status.Result = &metav1.Status{
		Status: metav1.StatusFailure, Code: http.StatusInternalServerError, Reason: metav1.StatusReasonInternalError,
		Message: err.Error(),
	}
	return status
}

func StatusBadRequest(err error) *v1beta1.AdmissionResponse {
	status := &v1beta1.AdmissionResponse{}
	status.Allowed = false
	status.Result = &metav1.Status{
		Status: metav1.StatusFailure, Code: http.StatusBadRequest, Reason: metav1.StatusReasonBadRequest,
		Message: err.Error(),
	}
	return status
}

func StatusForbidden(err error) *v1beta1.AdmissionResponse {
	status := &v1beta1.AdmissionResponse{}
	status.Allowed = false
	status.Result = &metav1.Status{
		Status: metav1.StatusFailure, Code: http.StatusForbidden, Reason: metav1.StatusReasonForbidden,
		Message: err.Error(),
	}
	return status
}
