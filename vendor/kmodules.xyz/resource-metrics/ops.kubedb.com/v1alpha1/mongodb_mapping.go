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

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
	RegisterOpsPathMapperToPlugins(&MongoDBOpsRequest{})
}

type MongoDBOpsRequest struct{}

var _ OpsPathMapper = (*MongoDBOpsRequest)(nil)

func (m *MongoDBOpsRequest) HorizontalPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.horizontalScaling.shard.shards":          "spec.shardTopology.shard.shards",
		"spec.horizontalScaling.shard.replicas":        "spec.shardTopology.shard.replicas",
		"spec.horizontalScaling.configServer.replicas": "spec.shardTopology.configServer.replicas",
		"spec.horizontalScaling.mongos.replicas":       "spec.shardTopology.mongos.replicas",
		"spec.horizontalScaling.hidden.replicas":       "spec.hidden.replicas",
		"spec.horizontalScaling.replicas":              "spec.replicas",
	}
}

func (m *MongoDBOpsRequest) VerticalPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.verticalScaling.standalone":   "spec.podTemplate.spec.resources",
		"spec.verticalScaling.replicaSet":   "spec.podTemplate.spec.resources",
		"spec.verticalScaling.mongos":       "spec.shardTopology.mongos.podTemplate.spec.resources",
		"spec.verticalScaling.configServer": "spec.shardTopology.configServer.podTemplate.spec.resources",
		"spec.verticalScaling.shard":        "spec.shardTopology.shard.podTemplate.spec.resources",
		"spec.verticalScaling.arbiter":      "spec.arbiter.podTemplate.spec.resources",
		"spec.verticalScaling.hidden":       "spec.hidden.podTemplate.spec.resources",
		"spec.verticalScaling.exporter":     "spec.monitor.prometheus.exporter.resources",
		//"spec.verticalScaling.coordinator":  "spec.coordinator.resources",
	}
}

func (m *MongoDBOpsRequest) VolumeExpansionPathMapping() map[OpsReqPath]ReferencedObjPath {
	return map[OpsReqPath]ReferencedObjPath{
		"spec.volumeExpansion.standalone":   "spec.storage.resources.requests.storage",
		"spec.volumeExpansion.replicaSet":   "spec.storage.resources.requests.storage",
		"spec.volumeExpansion.configServer": "spec.shardTopology.configServer.storage.resources.requests.storage",
		"spec.volumeExpansion.shard":        "spec.shardTopology.shard.storage.resources.requests.storage",
		"spec.volumeExpansion.hidden":       "spec.hidden.storage.resources.requests.storage",
	}
}

func (m *MongoDBOpsRequest) GetAppRefPath() []string {
	return []string{"spec", "databaseRef"}
}

func (m *MongoDBOpsRequest) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "ops.kubedb.com",
		Version: "v1alpha1",
		Kind:    "MongoDBOpsRequest",
	}
}
