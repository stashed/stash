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
	"kmodules.xyz/resource-metrics/api"

	batch "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
)

func init() {
	api.Register(batch.SchemeGroupVersion.WithKind("CronJob"), CronJob{}.ResourceCalculator())
}

type CronJob struct{}

func (r CronJob) ResourceCalculator() api.ResourceCalculator {
	return &api.ResourceCalculatorFuncs{
		AppRoles:               []api.PodRole{api.PodRoleDefault},
		RuntimeRoles:           []api.PodRole{api.PodRoleDefault},
		RoleReplicasFn:         r.roleReplicasFn,
		RoleResourceLimitsFn:   r.roleResourceFn(api.ResourceLimits),
		RoleResourceRequestsFn: r.roleResourceFn(api.ResourceRequests),
	}
}

func (_ CronJob) roleReplicasFn(obj map[string]interface{}) (api.ReplicaList, error) {
	return nil, nil
}

func (r CronJob) roleResourceFn(fn func(rr core.ResourceRequirements) core.ResourceList) func(obj map[string]interface{}) (map[api.PodRole]core.ResourceList, error) {
	return func(obj map[string]interface{}) (map[api.PodRole]core.ResourceList, error) {
		containers, err := api.AggregateContainerResources(obj, fn, api.AddResourceList, "spec", "jobTemplate", "spec", "template", "spec", "containers")
		if err != nil {
			return nil, err
		}
		initContainers, err := api.AggregateContainerResources(obj, fn, api.MaxResourceList, "spec", "jobTemplate", "spec", "template", "spec", "initContainers")
		if err != nil {
			return nil, err
		}
		return map[api.PodRole]core.ResourceList{
			api.PodRoleDefault: containers,
			api.PodRoleInit:    initContainers,
		}, nil
	}
}
