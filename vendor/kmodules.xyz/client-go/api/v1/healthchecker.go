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

package v1

// ReadonlyHealthCheckSpec defines attributes of the health check using only read-only checks
type ReadonlyHealthCheckSpec struct {
	// How often (in seconds) to perform the health check.
	// Default to 10 seconds. Minimum value is 1.
	// +optional
	// +kubebuilder:default=10
	PeriodSeconds *int32 `json:"periodSeconds,omitempty" protobuf:"varint,1,opt,name=periodSeconds"`
	// Number of seconds after which the probe times out.
	// Defaults to 10 second. Minimum value is 1.
	// It should be less than the periodSeconds.
	// +optional
	// +kubebuilder:default=10
	TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty" protobuf:"varint,2,opt,name=timeoutSeconds"`
	// Minimum consecutive failures for the health check to be considered failed after having succeeded.
	// Defaults to 1. Minimum value is 1.
	// +optional
	// +kubebuilder:default=1
	FailureThreshold *int32 `json:"failureThreshold,omitempty" protobuf:"varint,3,opt,name=failureThreshold"`
}

// HealthCheckSpec defines attributes of the health check
type HealthCheckSpec struct {
	ReadonlyHealthCheckSpec `json:",inline" protobuf:"bytes,1,opt,name=readonlyHealthCheckSpec"`
	// Whether to disable write check on database.
	// Defaults to false.
	// +optional
	DisableWriteCheck bool `json:"disableWriteCheck,omitempty" protobuf:"varint,2,opt,name=disableWriteCheck"`
}
