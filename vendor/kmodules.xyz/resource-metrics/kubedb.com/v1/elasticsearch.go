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
	"reflect"

	"kmodules.xyz/resource-metrics/api"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
	api.Register(schema.GroupVersionKind{
		Group:   "kubedb.com",
		Version: "v1",
		Kind:    "Elasticsearch",
	}, Elasticsearch{}.ResourceCalculator())
	api.Register(schema.GroupVersionKind{
		Group:   "gitops.kubedb.com",
		Version: "v1alpha1",
		Kind:    "Elasticsearch",
	}, Elasticsearch{}.ResourceCalculator())
}

type Elasticsearch struct{}

func (r Elasticsearch) ResourceCalculator() api.ResourceCalculator {
	return &api.ResourceCalculatorFuncs{
		AppRoles:               []api.PodRole{api.PodRoleDefault, api.PodRoleMaster, api.PodRoleIngest, api.PodRoleData, api.PodRoleDataContent, api.PodRoleDataCold, api.PodRoleDataHot, api.PodRoleDataWarm, api.PodRoleDataFrozen, api.PodRoleML, api.PodRoleTransform, api.PodRoleCoordinating},
		RuntimeRoles:           []api.PodRole{api.PodRoleDefault, api.PodRoleMaster, api.PodRoleIngest, api.PodRoleData, api.PodRoleDataContent, api.PodRoleDataCold, api.PodRoleDataHot, api.PodRoleDataWarm, api.PodRoleDataFrozen, api.PodRoleML, api.PodRoleTransform, api.PodRoleCoordinating, api.PodRoleExporter},
		RoleReplicasFn:         r.roleReplicasFn,
		ModeFn:                 r.modeFn,
		UsesTLSFn:              r.usesTLSFn,
		RoleResourceLimitsFn:   r.roleResourceFn(api.ResourceLimits),
		RoleResourceRequestsFn: r.roleResourceFn(api.ResourceRequests),
	}
}

func (r Elasticsearch) roleReplicasFn(obj map[string]any) (api.ReplicaList, error) {
	result := api.ReplicaList{}

	topology, found, err := unstructured.NestedMap(obj, "spec", "topology")
	if err != nil {
		return nil, err
	}
	if found && topology != nil {
		for role, roleSpec := range topology {
			roleReplicas, found, err := unstructured.NestedInt64(roleSpec.(map[string]any), "replicas")
			if err != nil {
				return nil, err
			}
			if found {
				result[api.PodRole(role)] = roleReplicas
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

func (r Elasticsearch) modeFn(obj map[string]any) (string, error) {
	topology, found, err := unstructured.NestedFieldNoCopy(obj, "spec", "topology")
	if err != nil {
		return "", err
	}
	if found && !reflect.ValueOf(topology).IsNil() {
		return DBModeDedicated, nil
	}
	return DBModeCombined, nil
}

func (r Elasticsearch) usesTLSFn(obj map[string]any) (bool, error) {
	_, found, err := unstructured.NestedFieldNoCopy(obj, "spec", "enableSSL")
	return found, err
}

func (r Elasticsearch) roleResourceFn(fn func(rr core.ResourceRequirements) core.ResourceList) func(obj map[string]any) (map[api.PodRole]api.PodInfo, error) {
	return func(obj map[string]any) (map[api.PodRole]api.PodInfo, error) {
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
			result := map[api.PodRole]api.PodInfo{}

			for role := range topology {
				rolePerReplicaResources, roleReplicas, err := api.AppNodeResourcesV2(topology, fn, ElasticsearchContainerName, role)
				if err != nil {
					return nil, err
				}
				replicas += roleReplicas
				result[api.PodRole(role)] = api.PodInfo{Resource: rolePerReplicaResources, Replicas: roleReplicas}
			}

			result[api.PodRoleExporter] = api.PodInfo{Resource: exporter, Replicas: replicas}
			return result, nil
		}

		// Elasticsearch Combined
		container, replicas, err := api.AppNodeResourcesV2(obj, fn, ElasticsearchContainerName, "spec")
		if err != nil {
			return nil, err
		}

		return map[api.PodRole]api.PodInfo{
			api.PodRoleDefault:  {Resource: container, Replicas: replicas},
			api.PodRoleExporter: {Resource: exporter, Replicas: replicas},
		}, nil
	}
}

type ElasticsearchNode struct {
	Replicas  *int64                         `json:"replicas,omitempty"`
	Resources core.ResourceRequirements      `json:"resources,omitempty"`
	Storage   core.PersistentVolumeClaimSpec `json:"storage,omitempty"`
}
