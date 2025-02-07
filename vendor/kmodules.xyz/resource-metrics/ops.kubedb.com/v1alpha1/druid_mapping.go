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
	RegisterOpsPathMapperToPlugins(&DruidOpsRequest{})
}

type DruidOpsRequest struct{}

var _ OpsPathMapper = (*DruidOpsRequest)(nil)

func (m *DruidOpsRequest) HorizontalPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.horizontalScaling.topology.coordinators":   "spec.topology.coordinators.replicas",
		"spec.horizontalScaling.topology.overloads":      "spec.topology.overloads.replicas",
		"spec.horizontalScaling.topology.middleManagers": "spec.topology.middleManagers.replicas",
		"spec.horizontalScaling.topology.historicals":    "spec.topology.historicals.replicas",
		"spec.horizontalScaling.topology.brokers":        "spec.topology.brokers.replicas",
		"spec.horizontalScaling.topology.routers":        "spec.topology.routers.replicas",
	}
}

func (m *DruidOpsRequest) VerticalPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.verticalScaling.topology.coordinators":   "spec.topology.coordinators.podTemplate.spec.resources",
		"spec.verticalScaling.topology.overloads":      "spec.topology.overloads.podTemplate.spec.resources",
		"spec.verticalScaling.topology.middleManagers": "spec.topology.middleManagers.podTemplate.spec.resources",
		"spec.verticalScaling.topology.historicals":    "spec.topology.historicals.podTemplate.spec.resources",
		"spec.verticalScaling.topology.brokers":        "spec.topology.brokers.podTemplate.spec.resources",
		"spec.verticalScaling.topology.routers":        "spec.topology.routers.podTemplate.spec.resources",
	}
}

func (m *DruidOpsRequest) VolumeExpansionPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.volumeExpansion.topology.middleManagers": "spec.topology.middleManagers.podTemplate.spec.storage.resources.requests.storage",
		"spec.volumeExpansion.topology.historicals":    "spec.topology.historicals.podTemplate.spec.storage.resources.requests.storage",
	}
}

func (m *DruidOpsRequest) GetAppRefPath() []string {
	return []string{"spec", "databaseRef"}
}

func (m *DruidOpsRequest) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "ops.kubedb.com",
		Version: "v1alpha1",
		Kind:    "DruidOpsRequest",
	}
}
