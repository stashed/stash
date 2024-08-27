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

import "k8s.io/apimachinery/pkg/runtime/schema"

func init() {
	RegisterOpsPathMapperToPlugins(&RedisSentinelOpsRequest{})
}

type RedisSentinelOpsRequest struct{}

var _ OpsPathMapper = (*RedisSentinelOpsRequest)(nil)

func (m *RedisSentinelOpsRequest) HorizontalPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.horizontalScaling.replicas": "spec.replicas",
	}
}

func (m *RedisSentinelOpsRequest) VerticalPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.verticalScaling.redissentinel": "spec.podTemplate.spec.resources",
		"spec.verticalScaling.exporter":      "spec.monitor.prometheus.exporter.resources",
		"spec.verticalScaling.coordinator":   "spec.coordinator.resources",
	}
}

func (m *RedisSentinelOpsRequest) VolumeExpansionPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.volumeExpansion.redissentinel": "spec.storage.resources.requests.storage",
	}
}

func (m *RedisSentinelOpsRequest) GetAppRefPath() []string {
	return []string{"spec", "databaseRef"}
}

func (m *RedisSentinelOpsRequest) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "ops.kubedb.com",
		Version: "v1alpha1",
		Kind:    "RedisSentinelOpsRequest",
	}
}
