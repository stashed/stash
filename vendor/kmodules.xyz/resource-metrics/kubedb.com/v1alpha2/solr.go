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

package v1alpha2

import (
	"fmt"
	"reflect"

	"kmodules.xyz/resource-metrics/api"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
	api.Register(schema.GroupVersionKind{
		Group:   "kubedb.com",
		Version: "v1alpha2",
		Kind:    "Solr",
	}, Solr{}.ResourceCalculator())
}

type Solr struct{}

func (r Solr) ResourceCalculator() api.ResourceCalculator {
	return &api.ResourceCalculatorFuncs{
		AppRoles:               []api.PodRole{api.PodRoleDefault},
		RuntimeRoles:           []api.PodRole{api.PodRoleDefault, api.PodRoleExporter},
		RoleReplicasFn:         r.roleReplicasFn,
		ModeFn:                 r.modeFn,
		UsesTLSFn:              r.usesTLSFn,
		RoleResourceLimitsFn:   r.roleResourceFn(api.ResourceLimits),
		RoleResourceRequestsFn: r.roleResourceFn(api.ResourceRequests),
	}
}

func (r Solr) roleReplicasFn(obj map[string]interface{}) (api.ReplicaList, error) {
	result := api.ReplicaList{}

	topology, found, err := unstructured.NestedMap(obj, "spec", "topology")
	if err != nil {
		return nil, err
	}

	if found && topology != nil {
		var replicas int64 = 0
		for role, roleSpec := range topology {
			roleReplicas, found, err := unstructured.NestedInt64(roleSpec.(map[string]interface{}), "replicas")
			if err != nil {
				return nil, err
			}
			if found {
				result[api.PodRole(role)] = roleReplicas
				replicas += roleReplicas
			} else {
				result[api.PodRole(role)] = 1
				replicas += 1
			}
		}
		result[api.PodRoleDefault] = replicas
	} else {
		// Combined mode
		replicas, found, err := unstructured.NestedInt64(obj, "spec", "replicas")
		if err != nil {
			return nil, fmt.Errorf("failed to read spec.replicas %v: %w", obj, err)
		}
		if !found {
			result[api.PodRoleDefault] = 1
		} else {
			result[api.PodRoleDefault] = replicas
		}
	}

	return result, nil
}

func (r Solr) modeFn(obj map[string]interface{}) (string, error) {
	topology, found, err := unstructured.NestedFieldNoCopy(obj, "spec", "topology")
	if err != nil {
		return "", err
	}
	if found && !reflect.ValueOf(topology).IsNil() {
		return DBModeDedicated, nil
	}
	return DBModeCombined, nil
}

func (r Solr) usesTLSFn(obj map[string]interface{}) (bool, error) {
	_, found, err := unstructured.NestedFieldNoCopy(obj, "spec", "enableSSL")
	return found, err
}

func (r Solr) roleResourceFn(fn func(rr core.ResourceRequirements) core.ResourceList) func(obj map[string]interface{}) (map[api.PodRole]core.ResourceList, error) {
	return func(obj map[string]interface{}) (map[api.PodRole]core.ResourceList, error) {
		exporter, err := api.ContainerResources(obj, fn, "spec", "monitor", "prometheus", "exporter")
		if err != nil {
			return nil, err
		}

		topology, found, err := unstructured.NestedMap(obj, "spec", "topology")
		if err != nil {
			return nil, err
		}
		if found && topology != nil {
			var replicas int64 = 0
			var totalResources core.ResourceList
			result := map[api.PodRole]core.ResourceList{}

			for role, roleSpec := range topology {
				rolePerReplicaResources, roleReplicas, err := api.AppNodeResourcesV2(roleSpec.(map[string]interface{}), fn, SolrContainerName)
				if err != nil {
					return nil, err
				}

				roleResources := api.MulResourceList(rolePerReplicaResources, roleReplicas)
				result[api.PodRole(role)] = roleResources
				totalResources = api.AddResourceList(totalResources, roleResources)
			}

			result[api.PodRoleDefault] = totalResources
			result[api.PodRoleExporter] = api.MulResourceList(exporter, replicas)
			return result, nil
		}

		// Solr Combined
		container, replicas, err := api.AppNodeResourcesV2(obj, fn, SolrContainerName, "spec")
		if err != nil {
			return nil, err
		}

		return map[api.PodRole]core.ResourceList{
			api.PodRoleDefault:  api.MulResourceList(container, replicas),
			api.PodRoleExporter: api.MulResourceList(exporter, replicas),
		}, nil
	}
}
