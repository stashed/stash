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
		Kind:    "Singlestore",
	}, Singlestore{}.ResourceCalculator())
}

type Singlestore struct{}

func (r Singlestore) ResourceCalculator() api.ResourceCalculator {
	return &api.ResourceCalculatorFuncs{
		AppRoles:               []api.PodRole{api.PodRoleDefault, api.PodRoleAggregator, api.PodRoleLeaf},
		RuntimeRoles:           []api.PodRole{api.PodRoleDefault, api.PodRoleAggregator, api.PodRoleLeaf, api.PodRoleExporter},
		RoleReplicasFn:         r.roleReplicasFn,
		ModeFn:                 r.modeFn,
		UsesTLSFn:              r.usesTLSFn,
		RoleResourceLimitsFn:   r.roleResourceFn(api.ResourceLimits),
		RoleResourceRequestsFn: r.roleResourceFn(api.ResourceRequests),
	}
}

func (r Singlestore) roleReplicasFn(obj map[string]interface{}) (api.ReplicaList, error) {
	result := api.ReplicaList{}

	topology, found, err := unstructured.NestedMap(obj, "spec", "topology")
	if err != nil {
		return nil, err
	}
	if found && topology != nil {
		// dedicated topology mode
		var replicas int64 = 0
		for role, roleSpec := range topology {
			roleReplicas, found, err := unstructured.NestedInt64(roleSpec.(map[string]interface{}), "replicas")
			if err != nil {
				return nil, err
			}
			if found {
				result[api.PodRole(role)] = roleReplicas
				replicas += roleReplicas
			}
		}
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

func (r Singlestore) modeFn(obj map[string]interface{}) (string, error) {
	topology, found, err := unstructured.NestedFieldNoCopy(obj, "spec", "topology")
	if err != nil {
		return "", err
	}
	if found && !reflect.ValueOf(topology).IsNil() {
		return DBModeDedicated, nil
	}
	return DBModeCombined, nil
}

func (r Singlestore) usesTLSFn(obj map[string]interface{}) (bool, error) {
	_, found, err := unstructured.NestedFieldNoCopy(obj, "spec", "tls")
	return found, err
}

func (r Singlestore) roleResourceFn(fn func(rr core.ResourceRequirements) core.ResourceList) func(obj map[string]interface{}) (map[api.PodRole]core.ResourceList, error) {
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
			aggregator, aggregatorReplicas, err := api.AppNodeResourcesV2(topology, fn, SinglestoreContainerName, "aggregator")
			if err != nil {
				return nil, err
			}

			leaf, leafReplicas, err := api.AppNodeResourcesV2(topology, fn, SinglestoreContainerName, "leaf")
			if err != nil {
				return nil, err
			}

			return map[api.PodRole]core.ResourceList{
				api.PodRoleAggregator: api.MulResourceList(aggregator, aggregatorReplicas),
				api.PodRoleLeaf:       api.MulResourceList(leaf, leafReplicas),
				api.PodRoleExporter:   api.MulResourceList(exporter, aggregatorReplicas+leafReplicas),
			}, nil
		}

		container, replicas, err := api.AppNodeResourcesV2(obj, fn, SinglestoreContainerName, "spec")
		if err != nil {
			return nil, err
		}

		return map[api.PodRole]core.ResourceList{
			api.PodRoleDefault:  api.MulResourceList(container, replicas),
			api.PodRoleExporter: api.MulResourceList(exporter, replicas),
		}, nil
	}
}
