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

import (
	"fmt"

	"kmodules.xyz/resource-metrics/api"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func init() {
	api.Register(apps.SchemeGroupVersion.WithKind("ReplicaSet"), ReplicaSet{}.ResourceCalculator())
}

type ReplicaSet struct{}

func (r ReplicaSet) ResourceCalculator() api.ResourceCalculator {
	return &api.ResourceCalculatorFuncs{
		AppRoles:               []api.PodRole{api.PodRoleDefault},
		RuntimeRoles:           []api.PodRole{api.PodRoleDefault},
		RoleReplicasFn:         r.roleReplicasFn,
		RoleResourceLimitsFn:   r.roleResourceFn(api.ResourceLimits),
		RoleResourceRequestsFn: r.roleResourceFn(api.ResourceRequests),
	}
}

func (_ ReplicaSet) roleReplicasFn(obj map[string]interface{}) (api.ReplicaList, error) {
	replicas, found, err := unstructured.NestedInt64(obj, "spec", "replicas")
	if err != nil {
		return nil, fmt.Errorf("failed to read spec.replicas %v: %w", obj, err)
	}
	if !found {
		return api.ReplicaList{api.PodRoleDefault: 1}, nil
	}
	return api.ReplicaList{api.PodRoleDefault: replicas}, nil
}

func (r ReplicaSet) roleResourceFn(fn func(rr core.ResourceRequirements) core.ResourceList) func(obj map[string]interface{}) (map[api.PodRole]core.ResourceList, error) {
	return func(obj map[string]interface{}) (map[api.PodRole]core.ResourceList, error) {
		rr, err := r.roleReplicasFn(obj)
		if err != nil {
			return nil, err
		}
		replicas := rr[api.PodRoleDefault]

		containers, err := api.AggregateContainerResources(obj, fn, api.AddResourceList, "spec", "template", "spec", "containers")
		if err != nil {
			return nil, err
		}
		initContainers, err := api.AggregateContainerResources(obj, fn, api.MaxResourceList, "spec", "template", "spec", "initContainers")
		if err != nil {
			return nil, err
		}
		return map[api.PodRole]core.ResourceList{
			api.PodRoleDefault: api.MulResourceList(containers, replicas),
			api.PodRoleInit:    api.MulResourceList(initContainers, replicas),
		}, nil
	}
}
