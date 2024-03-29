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

package v1

import (
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

// RollingDeploymentStrategyParamsApplyConfiguration represents an declarative configuration of the RollingDeploymentStrategyParams type for use
// with apply.
type RollingDeploymentStrategyParamsApplyConfiguration struct {
	UpdatePeriodSeconds *int64                           `json:"updatePeriodSeconds,omitempty"`
	IntervalSeconds     *int64                           `json:"intervalSeconds,omitempty"`
	TimeoutSeconds      *int64                           `json:"timeoutSeconds,omitempty"`
	MaxUnavailable      *intstr.IntOrString              `json:"maxUnavailable,omitempty"`
	MaxSurge            *intstr.IntOrString              `json:"maxSurge,omitempty"`
	Pre                 *LifecycleHookApplyConfiguration `json:"pre,omitempty"`
	Post                *LifecycleHookApplyConfiguration `json:"post,omitempty"`
}

// RollingDeploymentStrategyParamsApplyConfiguration constructs an declarative configuration of the RollingDeploymentStrategyParams type for use with
// apply.
func RollingDeploymentStrategyParams() *RollingDeploymentStrategyParamsApplyConfiguration {
	return &RollingDeploymentStrategyParamsApplyConfiguration{}
}

// WithUpdatePeriodSeconds sets the UpdatePeriodSeconds field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the UpdatePeriodSeconds field is set to the value of the last call.
func (b *RollingDeploymentStrategyParamsApplyConfiguration) WithUpdatePeriodSeconds(value int64) *RollingDeploymentStrategyParamsApplyConfiguration {
	b.UpdatePeriodSeconds = &value
	return b
}

// WithIntervalSeconds sets the IntervalSeconds field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the IntervalSeconds field is set to the value of the last call.
func (b *RollingDeploymentStrategyParamsApplyConfiguration) WithIntervalSeconds(value int64) *RollingDeploymentStrategyParamsApplyConfiguration {
	b.IntervalSeconds = &value
	return b
}

// WithTimeoutSeconds sets the TimeoutSeconds field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the TimeoutSeconds field is set to the value of the last call.
func (b *RollingDeploymentStrategyParamsApplyConfiguration) WithTimeoutSeconds(value int64) *RollingDeploymentStrategyParamsApplyConfiguration {
	b.TimeoutSeconds = &value
	return b
}

// WithMaxUnavailable sets the MaxUnavailable field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the MaxUnavailable field is set to the value of the last call.
func (b *RollingDeploymentStrategyParamsApplyConfiguration) WithMaxUnavailable(value intstr.IntOrString) *RollingDeploymentStrategyParamsApplyConfiguration {
	b.MaxUnavailable = &value
	return b
}

// WithMaxSurge sets the MaxSurge field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the MaxSurge field is set to the value of the last call.
func (b *RollingDeploymentStrategyParamsApplyConfiguration) WithMaxSurge(value intstr.IntOrString) *RollingDeploymentStrategyParamsApplyConfiguration {
	b.MaxSurge = &value
	return b
}

// WithPre sets the Pre field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Pre field is set to the value of the last call.
func (b *RollingDeploymentStrategyParamsApplyConfiguration) WithPre(value *LifecycleHookApplyConfiguration) *RollingDeploymentStrategyParamsApplyConfiguration {
	b.Pre = value
	return b
}

// WithPost sets the Post field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Post field is set to the value of the last call.
func (b *RollingDeploymentStrategyParamsApplyConfiguration) WithPost(value *LifecycleHookApplyConfiguration) *RollingDeploymentStrategyParamsApplyConfiguration {
	b.Post = value
	return b
}
