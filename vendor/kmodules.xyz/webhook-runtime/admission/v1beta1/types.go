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
	admission "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
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

type GetFunc func(namespace, name string) (runtime.Object, error)

type GetterFactory interface {
	New(config *rest.Config) (GetFunc, error)
}
