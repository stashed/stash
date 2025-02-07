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
	RegisterOpsPathMapperToPlugins(&SolrOpsRequest{})
}

type SolrOpsRequest struct{}

var _ OpsPathMapper = (*SolrOpsRequest)(nil)

func (m *SolrOpsRequest) HorizontalPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.horizontalScaling.node":        "spec.replicas",
		"spec.horizontalScaling.data":        "spec.topology.data.replicas",
		"spec.horizontalScaling.overseer":    "spec.topology.overseer.replicas",
		"spec.horizontalScaling.coordinator": "spec.topology.coordinator.replicas",
	}
}

func (m *SolrOpsRequest) VerticalPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.verticalScaling.node":        "spec.podTemplate.spec.resources",
		"spec.verticalScaling.data":        "spec.topology.data.podTemplate.spec.resources",
		"spec.verticalScaling.overseer":    "spec.topology.overseer.podTemplate.spec.resources",
		"spec.verticalScaling.coordinator": "spec.topology.coordinator.podTemplate.spec.resources",
	}
}

func (m *SolrOpsRequest) VolumeExpansionPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.volumeExpansion.node":        "spec.storage.resources.requests.storage",
		"spec.volumeExpansion.data":        "spec.topology.data.storage.resources.requests.storage",
		"spec.volumeExpansion.overseer":    "spec.topology.overseer.storage.resources.requests.storage",
		"spec.volumeExpansion.coordinator": "spec.topology.coordinator.storage.resources.requests.storage",
	}
}

func (m *SolrOpsRequest) GetAppRefPath() []string {
	return []string{"spec", "databaseRef"}
}

func (m *SolrOpsRequest) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "ops.kubedb.com",
		Version: "v1alpha1",
		Kind:    "SolrOpsRequest",
	}
}
