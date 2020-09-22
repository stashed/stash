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

package v1

import (
	"fmt"
	"strings"
)

const (
	KindPod                   = "Pod"
	KindDeployment            = "Deployment"
	KindReplicaSet            = "ReplicaSet"
	KindReplicationController = "ReplicationController"
	KindStatefulSet           = "StatefulSet"
	KindDaemonSet             = "DaemonSet"
	KindJob                   = "Job"
	KindCronJob               = "CronJob"
	KindDeploymentConfig      = "DeploymentConfig"

	ResourcePods                   = "pods"
	ResourceDeployments            = "deployments"
	ResourceReplicaSets            = "replicasets"
	ResourceReplicationControllers = "replicationsontrollers"
	ResourceStatefulSets           = "statefulsets"
	ResourceDaemonSets             = "daemonsets"
	ResourceJobs                   = "jobs"
	ResourceCronJobs               = "cronjobs"
	ResourceDeploymentConfigs      = "deploymentconfigs"

	ResourcePod                   = "pod"
	ResourceDeployment            = "deployment"
	ResourceReplicaSet            = "replicaset"
	ResourceReplicationController = "replicationsontroller"
	ResourceStatefulSet           = "statefulset"
	ResourceDaemonSet             = "daemonset"
	ResourceJob                   = "job"
	ResourceCronJob               = "cronjob"
	ResourceDeploymentConfig      = "deploymentconfig"
)

func Canonicalize(kind string) (string, error) {
	switch strings.ToLower(kind) {
	case "deployments", "deployment", "deploy":
		return KindDeployment, nil
	case "replicasets", "replicaset", "rs":
		return KindReplicaSet, nil
	case "replicationcontrollers", "replicationcontroller", "rc":
		return KindReplicationController, nil
	case "statefulsets", "statefulset":
		return KindStatefulSet, nil
	case "daemonsets", "daemonset", "ds":
		return KindDaemonSet, nil
	case "jobs", "job":
		return KindJob, nil
	case "cronjobs", "cronjob":
		return KindCronJob, nil
	case "deploymentconfigs", "deploymentconfig":
		return KindDeploymentConfig, nil
	default:
		return "", fmt.Errorf(`unrecognized workload "Kind" %v`, kind)
	}
}
