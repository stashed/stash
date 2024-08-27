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
	RegisterOpsPathMapperToPlugins(&ElasticsearchOpsRequest{})
}

type ElasticsearchOpsRequest struct{}

var _ OpsPathMapper = (*ElasticsearchOpsRequest)(nil)

func (m *ElasticsearchOpsRequest) HorizontalPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.horizontalScaling.node":                  "spec.replicas",
		"spec.horizontalScaling.topology.master":       "spec.topology.master.replicas",
		"spec.horizontalScaling.topology.ingest":       "spec.topology.ingest.replicas",
		"spec.horizontalScaling.topology.data":         "spec.topology.data.replicas",
		"spec.horizontalScaling.topology.dataContent":  "spec.topology.dataContent.replicas",
		"spec.horizontalScaling.topology.dataHot":      "spec.topology.dataHot.replicas",
		"spec.horizontalScaling.topology.dataWarm":     "spec.topology.dataWarm.replicas",
		"spec.horizontalScaling.topology.dataCold":     "spec.topology.dataCold.replicas",
		"spec.horizontalScaling.topology.dataFrozen":   "spec.topology.dataFrozen.replicas",
		"spec.horizontalScaling.topology.ml":           "spec.topology.ml.replicas",
		"spec.horizontalScaling.topology.transform":    "spec.topology.transform.replicas",
		"spec.horizontalScaling.topology.coordinating": "spec.topology.coordinating.replicas",
	}
}

func (m *ElasticsearchOpsRequest) VerticalPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.verticalScaling.node":                  "spec.resources",
		"spec.verticalScaling.exporter":              "spec.topology.exporter.resources",
		"spec.verticalScaling.topology.master":       "spec.topology.master.resources",
		"spec.verticalScaling.topology.ingest":       "spec.topology.ingest.resources",
		"spec.verticalScaling.topology.data":         "spec.topology.data.resources",
		"spec.verticalScaling.topology.dataContent":  "spec.topology.dataContent.resources",
		"spec.verticalScaling.topology.dataHot":      "spec.topology.dataHot.resources",
		"spec.verticalScaling.topology.dataWarm":     "spec.topology.dataWarm.resources",
		"spec.verticalScaling.topology.dataCold":     "spec.topology.dataCold.resources",
		"spec.verticalScaling.topology.dataFrozen":   "spec.topology.dataFrozen.resources",
		"spec.verticalScaling.topology.ml":           "spec.topology.ml.resources",
		"spec.verticalScaling.topology.transform":    "spec.topology.transform.resources",
		"spec.verticalScaling.topology.coordinating": "spec.topology.coordinating.resources",
	}
}

func (m *ElasticsearchOpsRequest) VolumeExpansionPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.volumeExpansion.node":                  "spec.storage.resources.requests.storage",
		"spec.volumeExpansion.topology.master":       "spec.topology.master.storage.resources.requests.storage",
		"spec.volumeExpansion.topology.ingest":       "spec.topology.ingest.storage.resources.requests.storage",
		"spec.volumeExpansion.topology.data":         "spec.topology.data.storage.resources.requests.storage",
		"spec.volumeExpansion.topology.dataContent":  "spec.topology.dataContent.storage.resources.requests.storage",
		"spec.volumeExpansion.topology.dataHot":      "spec.topology.dataHot.storage.resources.requests.storage",
		"spec.volumeExpansion.topology.dataWarm":     "spec.topology.dataWarm.storage.resources.requests.storage",
		"spec.volumeExpansion.topology.dataCold":     "spec.topology.dataCold.storage.resources.requests.storage",
		"spec.volumeExpansion.topology.dataFrozen":   "spec.topology.dataFrozen.storage.resources.requests.storage",
		"spec.volumeExpansion.topology.ml":           "spec.topology.ml.storage.resources.requests.storage",
		"spec.volumeExpansion.topology.transform":    "spec.topology.transform.storage.resources.requests.storage",
		"spec.volumeExpansion.topology.coordinating": "spec.topology.coordinating.storage.resources.requests.storage",
	}
}

func (m *ElasticsearchOpsRequest) GetAppRefPath() []string {
	return []string{"spec", "databaseRef"}
}

func (m *ElasticsearchOpsRequest) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "ops.kubedb.com",
		Version: "v1alpha1",
		Kind:    "ElasticsearchOpsRequest",
	}
}
