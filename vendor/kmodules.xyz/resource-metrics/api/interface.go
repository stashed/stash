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

package api

import (
	core "k8s.io/api/core/v1"
)

type ResourceCalculator interface {
	Replicas(obj map[string]interface{}) (int64, error)
	RoleReplicas(obj map[string]interface{}) (ReplicaList, error)

	Mode(obj map[string]interface{}) (string, error)
	UsesTLS(obj map[string]interface{}) (bool, error)

	TotalResourceLimits(obj map[string]interface{}) (core.ResourceList, error)
	TotalResourceRequests(obj map[string]interface{}) (core.ResourceList, error)

	AppResourceLimits(obj map[string]interface{}) (core.ResourceList, error)
	AppResourceRequests(obj map[string]interface{}) (core.ResourceList, error)

	RoleResourceLimits(obj map[string]interface{}) (map[PodRole]core.ResourceList, error)
	RoleResourceRequests(obj map[string]interface{}) (map[PodRole]core.ResourceList, error)
}

type ResourceCalculatorFuncs struct {
	// Resources used by the main application (eg, database) containers
	AppRoles []PodRole

	// usually AppRoles + Exporter + Any custom sidecar (label selector etc.) that is used at runtime
	// Must NOT include init container resources
	RuntimeRoles []PodRole

	RoleReplicasFn         func(obj map[string]interface{}) (ReplicaList, error)
	ModeFn                 func(obj map[string]interface{}) (string, error)
	UsesTLSFn              func(obj map[string]interface{}) (bool, error)
	RoleResourceLimitsFn   func(obj map[string]interface{}) (map[PodRole]core.ResourceList, error)
	RoleResourceRequestsFn func(obj map[string]interface{}) (map[PodRole]core.ResourceList, error)
}

var _ ResourceCalculator = &ResourceCalculatorFuncs{}

func (c ResourceCalculatorFuncs) Replicas(obj map[string]interface{}) (int64, error) {
	replicas, err := c.RoleReplicas(obj)
	if err != nil {
		return 0, err
	}
	var cnt int64 = 0
	for _, role := range c.AppRoles {
		cnt += replicas[role]
	}
	return cnt, nil
}

func (c ResourceCalculatorFuncs) RoleReplicas(obj map[string]interface{}) (ReplicaList, error) {
	return c.RoleReplicasFn(obj)
}

func (c ResourceCalculatorFuncs) Mode(obj map[string]interface{}) (string, error) {
	if c.ModeFn != nil {
		return c.ModeFn(obj)
	}
	return "", nil
}

func (c ResourceCalculatorFuncs) UsesTLS(obj map[string]interface{}) (bool, error) {
	if c.UsesTLSFn != nil {
		return c.UsesTLSFn(obj)
	}
	return false, nil
}

func (c ResourceCalculatorFuncs) TotalResourceLimits(obj map[string]interface{}) (core.ResourceList, error) {
	rr, err := c.RoleResourceLimits(obj)
	if err != nil {
		return nil, err
	}
	return MaxResourceList(
		ResourceListForRoles(rr, c.RuntimeRoles),
		ResourceListForRoles(rr, []PodRole{PodRoleInit}),
	), nil
}

func (c ResourceCalculatorFuncs) TotalResourceRequests(obj map[string]interface{}) (core.ResourceList, error) {
	rr, err := c.RoleResourceRequests(obj)
	if err != nil {
		return nil, err
	}
	return MaxResourceList(
		ResourceListForRoles(rr, c.RuntimeRoles),
		ResourceListForRoles(rr, []PodRole{PodRoleInit}),
	), nil
}

func (c ResourceCalculatorFuncs) AppResourceLimits(obj map[string]interface{}) (core.ResourceList, error) {
	rr, err := c.RoleResourceLimits(obj)
	if err != nil {
		return nil, err
	}
	return ResourceListForRoles(rr, c.AppRoles), nil
}

func (c ResourceCalculatorFuncs) AppResourceRequests(obj map[string]interface{}) (core.ResourceList, error) {
	rr, err := c.RoleResourceRequests(obj)
	if err != nil {
		return nil, err
	}
	return ResourceListForRoles(rr, c.AppRoles), nil
}

func (c ResourceCalculatorFuncs) RoleResourceLimits(obj map[string]interface{}) (map[PodRole]core.ResourceList, error) {
	return c.RoleResourceLimitsFn(obj)
}

func (c ResourceCalculatorFuncs) RoleResourceRequests(obj map[string]interface{}) (map[PodRole]core.ResourceList, error) {
	return c.RoleResourceRequestsFn(obj)
}
