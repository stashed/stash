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

	"kmodules.xyz/resource-metrics/api"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
	api.Register(schema.GroupVersionKind{
		Group:   "kubedb.com",
		Version: "v1alpha2",
		Kind:    "Oracle",
	}, Oracle{}.ResourceCalculator())
	api.Register(schema.GroupVersionKind{
		Group:   "gitops.kubedb.com",
		Version: "v1alpha1",
		Kind:    "Oracle",
	}, Oracle{}.ResourceCalculator())
}

type Oracle struct{}

func (r Oracle) ResourceCalculator() api.ResourceCalculator {
	return &api.ResourceCalculatorFuncs{
		AppRoles:               []api.PodRole{api.PodRoleDefault},
		RuntimeRoles:           []api.PodRole{api.PodRoleDefault, api.PodRoleSidecar, api.PodRoleObserver},
		RoleReplicasFn:         r.roleReplicasFn,
		ModeFn:                 r.modeFn,
		UsesTLSFn:              r.usesTLSFn,
		RoleResourceLimitsFn:   r.roleResourceFn(api.ResourceLimits),
		RoleResourceRequestsFn: r.roleResourceFn(api.ResourceRequests),
	}
}

func (r Oracle) roleReplicasFn(obj map[string]any) (api.ReplicaList, error) {
	replicas, found, err := unstructured.NestedInt64(obj, "spec", "replicas")
	if err != nil {
		return nil, fmt.Errorf("failed to read spec.replicas %v: %w", obj, err)
	}
	if !found {
		return api.ReplicaList{api.PodRoleDefault: 1}, nil
	}
	return api.ReplicaList{api.PodRoleDefault: replicas}, nil
}

func (r Oracle) modeFn(obj map[string]any) (string, error) {
	replicas, _, err := unstructured.NestedInt64(obj, "spec", "replicas")
	if err != nil {
		return "", err
	}
	if replicas > 1 {
		return DBModeCluster, nil
	}
	return DBModeStandalone, nil
}

func (r Oracle) usesTLSFn(obj map[string]any) (bool, error) {
	_, found, err := unstructured.NestedFieldNoCopy(obj, "spec", "tls")
	return found, err
}

func (r Oracle) roleResourceFn(fn func(rr core.ResourceRequirements) core.ResourceList) func(obj map[string]any) (map[api.PodRole]api.PodInfo, error) {
	return func(obj map[string]any) (map[api.PodRole]api.PodInfo, error) {
		container, replicas, err := api.AppNodeResourcesV2(obj, fn, OracleContainerName, "spec")
		if err != nil {
			return nil, err
		}

		ret := map[api.PodRole]api.PodInfo{
			api.PodRoleDefault: {Resource: container, Replicas: replicas},
		}

		if replicas > 1 {
			sidecar, err := api.SidecarNodeResourcesV2(obj, fn, OracleSidecarContainerName, "spec")
			if err != nil {
				return nil, err
			}
			ret[api.PodRoleSidecar] = api.PodInfo{Resource: sidecar, Replicas: replicas}

			observer, err := api.SidecarNodeResourcesV2(obj, fn, OracleObserverContainerName, "spec", "dataGuard", "observer")
			if err != nil {
				return nil, err
			}
			ret[api.PodRoleObserver] = api.PodInfo{Resource: observer, Replicas: 1}
		}
		return ret, nil
	}
}
