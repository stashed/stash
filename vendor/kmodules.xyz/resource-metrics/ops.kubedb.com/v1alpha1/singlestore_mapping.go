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
	RegisterOpsPathMapperToPlugins(&SinglestoreOpsRequest{})
}

type SinglestoreOpsRequest struct{}

var _ OpsPathMapper = (*SinglestoreOpsRequest)(nil)

func (m *SinglestoreOpsRequest) HorizontalPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.horizontalScaling.aggregator": "spec.topology.aggregator.replicas",
		"spec.horizontalScaling.leaf":       "spec.topology.leaf.replicas",
	}
}

func (m *SinglestoreOpsRequest) VerticalPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.verticalScaling.node":        "spec.podTemplate.spec.resources",
		"spec.verticalScaling.aggregator":  "spec.topology.aggregator.podTemplate.spec.resources",
		"spec.verticalScaling.leaf":        "spec.topology.leaf.podTemplate.spec.resources",
		"spec.verticalScaling.coordinator": "spec.coordinator.resources",
	}
}

func (m *SinglestoreOpsRequest) VolumeExpansionPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.volumeExpansion.node":       "spec.storage.resources.requests.storage",
		"spec.volumeExpansion.aggregator": "spec.topology.aggregator.storage.resources.requests.storage",
		"spec.volumeExpansion.leaf":       "spec.topology.leaf.storage.resources.requests.storage",
	}
}

func (m *SinglestoreOpsRequest) GetAppRefPath() []string {
	return []string{"spec", "databaseRef"}
}

func (m *SinglestoreOpsRequest) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "ops.kubedb.com",
		Version: "v1alpha1",
		Kind:    "SinglestoreOpsRequest",
	}
}
