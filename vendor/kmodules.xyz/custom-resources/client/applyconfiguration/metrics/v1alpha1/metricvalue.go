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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

// MetricValueApplyConfiguration represents an declarative configuration of the MetricValue type for use
// with apply.
type MetricValueApplyConfiguration struct {
	Value               *float64 `json:"value,omitempty"`
	ValueFromPath       *string  `json:"valueFromPath,omitempty"`
	ValueFromExpression *string  `json:"valueFromExpression,omitempty"`
}

// MetricValueApplyConfiguration constructs an declarative configuration of the MetricValue type for use with
// apply.
func MetricValue() *MetricValueApplyConfiguration {
	return &MetricValueApplyConfiguration{}
}

// WithValue sets the Value field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Value field is set to the value of the last call.
func (b *MetricValueApplyConfiguration) WithValue(value float64) *MetricValueApplyConfiguration {
	b.Value = &value
	return b
}

// WithValueFromPath sets the ValueFromPath field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ValueFromPath field is set to the value of the last call.
func (b *MetricValueApplyConfiguration) WithValueFromPath(value string) *MetricValueApplyConfiguration {
	b.ValueFromPath = &value
	return b
}

// WithValueFromExpression sets the ValueFromExpression field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ValueFromExpression field is set to the value of the last call.
func (b *MetricValueApplyConfiguration) WithValueFromExpression(value string) *MetricValueApplyConfiguration {
	b.ValueFromExpression = &value
	return b
}
