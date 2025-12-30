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
	"slices"

	core "k8s.io/api/core/v1"
)

type ResourceCalculator interface {
	GetRoleResourceLimitsFn() func(obj map[string]any) (map[PodRole]PodInfo, error)
	GetRoleResourceRequestsFn() func(obj map[string]any) (map[PodRole]PodInfo, error)

	Replicas(obj map[string]any) (int64, error)
	RoleReplicas(obj map[string]any) (ReplicaList, error)

	Mode(obj map[string]any) (string, error)
	UsesTLS(obj map[string]any) (bool, error)

	TotalResourceLimits(obj map[string]any) (core.ResourceList, error)
	TotalResourceRequests(obj map[string]any) (core.ResourceList, error)

	AppResourceLimits(obj map[string]any) (core.ResourceList, error)
	AppResourceRequests(obj map[string]any) (core.ResourceList, error)

	RoleResourceLimits(obj map[string]any) (map[PodRole]core.ResourceList, error)
	RoleResourceRequests(obj map[string]any) (map[PodRole]core.ResourceList, error)

	PodResourceRequests(obj map[string]any) (core.ResourceList, error)
	PodResourceLimits(obj map[string]any) (core.ResourceList, error)
}

type ResourceCalculatorFuncs struct {
	// Resources used by the main application (eg, database) containers
	AppRoles []PodRole

	// usually AppRoles + Exporter + Any custom sidecar (label selector etc.) that is used at runtime
	// Must NOT include init container resources
	RuntimeRoles []PodRole

	RoleReplicasFn         func(obj map[string]any) (ReplicaList, error)
	ModeFn                 func(obj map[string]any) (string, error)
	UsesTLSFn              func(obj map[string]any) (bool, error)
	RoleResourceLimitsFn   func(obj map[string]any) (map[PodRole]PodInfo, error)
	RoleResourceRequestsFn func(obj map[string]any) (map[PodRole]PodInfo, error)
}

var _ ResourceCalculator = &ResourceCalculatorFuncs{}

func (c ResourceCalculatorFuncs) GetRoleResourceLimitsFn() func(obj map[string]any) (map[PodRole]PodInfo, error) {
	return c.RoleResourceLimitsFn
}

func (c ResourceCalculatorFuncs) GetRoleResourceRequestsFn() func(obj map[string]any) (map[PodRole]PodInfo, error) {
	return c.RoleResourceRequestsFn
}

func (c ResourceCalculatorFuncs) Replicas(obj map[string]any) (int64, error) {
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

func (c ResourceCalculatorFuncs) RoleReplicas(obj map[string]any) (ReplicaList, error) {
	return c.RoleReplicasFn(obj)
}

func (c ResourceCalculatorFuncs) Mode(obj map[string]any) (string, error) {
	if c.ModeFn != nil {
		return c.ModeFn(obj)
	}
	return "", nil
}

func (c ResourceCalculatorFuncs) UsesTLS(obj map[string]any) (bool, error) {
	if c.UsesTLSFn != nil {
		return c.UsesTLSFn(obj)
	}
	return false, nil
}

func (c ResourceCalculatorFuncs) TotalResourceLimits(obj map[string]any) (core.ResourceList, error) {
	rr, err := c.RoleResourceLimits(obj)
	if err != nil {
		return nil, err
	}
	return MaxResourceList(
		ResourceListForRoles(rr, c.RuntimeRoles),
		ResourceListForRoles(rr, []PodRole{PodRoleInit}),
	), nil
}

func (c ResourceCalculatorFuncs) TotalResourceRequests(obj map[string]any) (core.ResourceList, error) {
	rr, err := c.RoleResourceRequests(obj)
	if err != nil {
		return nil, err
	}
	return MaxResourceList(
		ResourceListForRoles(rr, c.RuntimeRoles),
		ResourceListForRoles(rr, []PodRole{PodRoleInit}),
	), nil
}

func (c ResourceCalculatorFuncs) AppResourceLimits(obj map[string]any) (core.ResourceList, error) {
	rr, err := c.RoleResourceLimits(obj)
	if err != nil {
		return nil, err
	}
	return ResourceListForRoles(rr, c.AppRoles), nil
}

func (c ResourceCalculatorFuncs) AppResourceRequests(obj map[string]any) (core.ResourceList, error) {
	rr, err := c.RoleResourceRequests(obj)
	if err != nil {
		return nil, err
	}
	return ResourceListForRoles(rr, c.AppRoles), nil
}

func (c ResourceCalculatorFuncs) RoleResourceLimits(obj map[string]any) (map[PodRole]core.ResourceList, error) {
	ret := make(map[PodRole]core.ResourceList)
	rr, err := c.RoleResourceLimitsFn(obj)
	if err != nil {
		return nil, err
	}
	for role, info := range rr {
		ret[role] = MulResourceList(info.Resource, info.Replicas)
	}
	return ret, nil
}

func (c ResourceCalculatorFuncs) RoleResourceRequests(obj map[string]any) (map[PodRole]core.ResourceList, error) {
	ret := make(map[PodRole]core.ResourceList)
	rr, err := c.RoleResourceRequestsFn(obj)
	if err != nil {
		return nil, err
	}
	for role, info := range rr {
		ret[role] = MulResourceList(info.Resource, info.Replicas)
	}
	return ret, nil
}

func (c ResourceCalculatorFuncs) PodResourceLimits(obj map[string]any) (core.ResourceList, error) {
	rl, err := c.RoleResourceLimitsFn(obj)
	if err != nil {
		return nil, err
	}
	return c.calcForPod(rl), nil
}

func (c ResourceCalculatorFuncs) PodResourceRequests(obj map[string]any) (core.ResourceList, error) {
	rr, err := c.RoleResourceRequestsFn(obj)
	if err != nil {
		return nil, err
	}
	return c.calcForPod(rr), nil
}

func (c ResourceCalculatorFuncs) calcForPod(roleInfoMap map[PodRole]PodInfo) core.ResourceList {
	mx := core.ResourceList{}
	extraRoles := c.getExtraRoles()
	for _, role := range c.AppRoles {
		if _, exist := roleInfoMap[role]; !exist {
			continue
		}
		res := roleInfoMap[role].Resource
		for _, extraRole := range extraRoles {
			if _, exist := roleInfoMap[extraRole]; !exist {
				continue
			}
			extraRes := roleInfoMap[extraRole].Resource
			res = AddResourceList(res, extraRes)
		}
		if initInfo, exist := roleInfoMap[PodRoleInit]; exist {
			res = MaxResourceList(res, initInfo.Resource)
		}
		mx = MaxResourceList(res, mx)
	}
	return mx
}

func (c ResourceCalculatorFuncs) getExtraRoles() []PodRole {
	roles := make([]PodRole, 0)
	for _, role := range c.RuntimeRoles {
		if !slices.Contains(c.AppRoles, role) {
			roles = append(roles, role)
		}
	}
	return roles
}
