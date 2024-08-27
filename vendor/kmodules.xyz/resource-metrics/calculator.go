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

package resourcemetrics

import (
	"kmodules.xyz/resource-metrics/api"
	_ "kmodules.xyz/resource-metrics/apps/v1"
	_ "kmodules.xyz/resource-metrics/batch/v1"
	_ "kmodules.xyz/resource-metrics/batch/v1beta1"
	_ "kmodules.xyz/resource-metrics/core/v1"
	_ "kmodules.xyz/resource-metrics/kafka.kubedb.com/v1alpha1"
	_ "kmodules.xyz/resource-metrics/kubedb.com/v1alpha2"
	_ "kmodules.xyz/resource-metrics/kubevault.com/v1alpha2"
	_ "kmodules.xyz/resource-metrics/ops.kubedb.com/v1alpha1"

	core "k8s.io/api/core/v1"
)

func Replicas(obj map[string]interface{}) (int64, error) {
	c, err := api.Load(obj)
	if err != nil {
		return 0, err
	}
	return c.Replicas(obj)
}

func RoleReplicas(obj map[string]interface{}) (api.ReplicaList, error) {
	c, err := api.Load(obj)
	if err != nil {
		return nil, err
	}
	return c.RoleReplicas(obj)
}

func Mode(obj map[string]interface{}) (string, error) {
	c, err := api.Load(obj)
	if err != nil {
		return "", err
	}
	return c.Mode(obj)
}

func UsesTLS(obj map[string]interface{}) (bool, error) {
	c, err := api.Load(obj)
	if err != nil {
		return false, err
	}
	return c.UsesTLS(obj)
}

func TotalResourceLimits(obj map[string]interface{}) (core.ResourceList, error) {
	c, err := api.Load(obj)
	if err != nil {
		return nil, err
	}
	return c.TotalResourceLimits(obj)
}

func TotalResourceRequests(obj map[string]interface{}) (core.ResourceList, error) {
	c, err := api.Load(obj)
	if err != nil {
		return nil, err
	}
	return c.TotalResourceRequests(obj)
}

func AppResourceLimits(obj map[string]interface{}) (core.ResourceList, error) {
	c, err := api.Load(obj)
	if err != nil {
		return nil, err
	}
	return c.AppResourceLimits(obj)
}

func AppResourceRequests(obj map[string]interface{}) (core.ResourceList, error) {
	c, err := api.Load(obj)
	if err != nil {
		return nil, err
	}
	return c.AppResourceRequests(obj)
}

func RoleResourceLimits(obj map[string]interface{}) (map[api.PodRole]core.ResourceList, error) {
	c, err := api.Load(obj)
	if err != nil {
		return nil, err
	}
	return c.RoleResourceLimits(obj)
}

func RoleResourceRequests(obj map[string]interface{}) (map[api.PodRole]core.ResourceList, error) {
	c, err := api.Load(obj)
	if err != nil {
		return nil, err
	}
	return c.RoleResourceRequests(obj)
}
