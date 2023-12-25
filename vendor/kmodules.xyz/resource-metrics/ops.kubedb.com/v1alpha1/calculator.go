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

package v1alpha1

import (
	"kmodules.xyz/resource-metrics/api"

	core "k8s.io/api/core/v1"
)

type OpsResourceCalculator struct{}

func (r OpsResourceCalculator) ResourceCalculator() api.ResourceCalculator {
	return &api.ResourceCalculatorFuncs{
		AppRoles:               []api.PodRole{api.PodRoleDefault},
		RuntimeRoles:           []api.PodRole{api.PodRoleDefault, api.PodRoleExporter},
		RoleReplicasFn:         r.roleReplicasFn,
		ModeFn:                 r.modeFn,
		UsesTLSFn:              r.usesTLSFn,
		RoleResourceLimitsFn:   r.roleResourceLimitsFn,
		RoleResourceRequestsFn: r.roleResourceRequestsFn,
	}
}

func (r OpsResourceCalculator) roleReplicasFn(opsObj map[string]interface{}) (api.ReplicaList, error) {
	scaledObject, err := GetScaledObject(opsObj)
	if err != nil {
		return nil, err
	}

	c, err := api.Load(scaledObject)
	if err != nil {
		return nil, err
	}
	return c.RoleReplicas(scaledObject)
}

func (r OpsResourceCalculator) modeFn(opsObj map[string]interface{}) (string, error) {
	scaledObject, err := GetScaledObject(opsObj)
	if err != nil {
		return "", err
	}

	c, err := api.Load(scaledObject)
	if err != nil {
		return "", err
	}
	return c.Mode(scaledObject)
}

func (r OpsResourceCalculator) usesTLSFn(opsObj map[string]interface{}) (bool, error) {
	scaledObject, err := GetScaledObject(opsObj)
	if err != nil {
		return false, err
	}

	c, err := api.Load(scaledObject)
	if err != nil {
		return false, err
	}
	return c.UsesTLS(scaledObject)
}

func (r OpsResourceCalculator) roleResourceLimitsFn(opsObj map[string]interface{}) (map[api.PodRole]core.ResourceList, error) {
	scaledObject, err := GetScaledObject(opsObj)
	if err != nil {
		return nil, err
	}

	c, err := api.Load(scaledObject)
	if err != nil {
		return nil, err
	}
	return c.RoleResourceLimits(scaledObject)
}

func (r OpsResourceCalculator) roleResourceRequestsFn(opsObj map[string]interface{}) (map[api.PodRole]core.ResourceList, error) {
	scaledObject, err := GetScaledObject(opsObj)
	if err != nil {
		return nil, err
	}

	c, err := api.Load(scaledObject)
	if err != nil {
		return nil, err
	}
	return c.RoleResourceRequests(scaledObject)
}
