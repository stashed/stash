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

package api

import (
	"errors"

	core "k8s.io/api/core/v1"
)

type PodRole string

const (
	PodRoleDefault          PodRole = ""
	PodRoleInit             PodRole = "init"
	PodRoleRouter           PodRole = "router"
	PodRoleExporter         PodRole = "exporter"
	PodRoleTotalShard       PodRole = "total_shard"
	PodRoleShard            PodRole = "shard"
	PodRoleReplicasPerShard PodRole = "replicas_per_shard"
	PodRoleConfigServer     PodRole = "config_server"
	PodRoleMongos           PodRole = "mongos"
	PodRoleAggregator       PodRole = "aggregator"
	PodRoleLeaf             PodRole = "leaf"
	PodRoleMaster           PodRole = "master"
	PodRoleData             PodRole = "data"
	PodRoleIngest           PodRole = "ingest"
	PodRoleDataContent      PodRole = "dataContent"
	PodRoleDataHot          PodRole = "dataHot"
	PodRoleDataWarm         PodRole = "dataWarm"
	PodRoleDataCold         PodRole = "dataCold"
	PodRoleDataFrozen       PodRole = "dataFrozen"
	PodRoleML               PodRole = "ml"
	PodRoleTransform        PodRole = "transform"
	PodRoleCoordinating     PodRole = "coordinating"
	PodRoleOverseer         PodRole = "overseer"
	PodRoleCoordinator      PodRole = "coordinator"
	PodRoleCoordinators     PodRole = "coordinators"
	PodRoleBroker           PodRole = "broker"
	PodRoleBrokers          PodRole = "brokers"
	PodRoleController       PodRole = "controller"
	PodRoleCombined         PodRole = "combined"
	PodRoleOverlords        PodRole = "overlords"
	PodRoleMiddleManagers   PodRole = "middleManagers"
	PodRoleHistoricals      PodRole = "historicals"
	PodRoleRouters          PodRole = "routers"
)

var ErrMissingRefObject = errors.New("referenced object not found")

type ReplicaList map[PodRole]int64

type PodInfo struct {
	Resource core.ResourceList
	Replicas int64
}
