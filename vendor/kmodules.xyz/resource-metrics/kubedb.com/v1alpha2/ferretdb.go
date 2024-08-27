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
		Kind:    "FerretDB",
	}, FerretDB{}.ResourceCalculator())
}

type FerretDB struct{}

func (r FerretDB) ResourceCalculator() api.ResourceCalculator {
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

func (r FerretDB) roleReplicasFn(obj map[string]interface{}) (api.ReplicaList, error) {
	replicas, found, err := unstructured.NestedInt64(obj, "spec", "replicas")
	if err != nil {
		return nil, fmt.Errorf("failed to read spec.replicas %v: %w", obj, err)
	}
	if !found {
		return api.ReplicaList{api.PodRoleDefault: 1}, nil
	}
	return api.ReplicaList{api.PodRoleDefault: replicas}, nil
}

func (r FerretDB) modeFn(obj map[string]interface{}) (string, error) {
	replicas, _, err := unstructured.NestedInt64(obj, "spec", "replicas")
	if err != nil {
		return "", err
	}
	if replicas > 1 {
		return DBModeCluster, nil
	}
	return DBModeStandalone, nil
}

func (r FerretDB) usesTLSFn(obj map[string]interface{}) (bool, error) {
	_, found, err := unstructured.NestedFieldNoCopy(obj, "spec", "tls")
	return found, err
}

func (r FerretDB) roleResourceFn(fn func(rr core.ResourceRequirements) core.ResourceList) func(obj map[string]interface{}) (map[api.PodRole]core.ResourceList, error) {
	return func(obj map[string]interface{}) (map[api.PodRole]core.ResourceList, error) {
		container, replicas, err := api.AppNodeResourcesV2(obj, fn, FerretDBContainerName, "spec")
		if err != nil {
			return nil, err
		}

		exporter, err := api.ContainerResources(obj, fn, "spec", "monitor", "prometheus", "exporter")
		if err != nil {
			return nil, err
		}
		return map[api.PodRole]core.ResourceList{
			api.PodRoleDefault:  api.MulResourceList(container, replicas),
			api.PodRoleExporter: api.MulResourceList(exporter, replicas),
		}, nil
	}
}
