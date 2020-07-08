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

package apis

const (
	StashKey   = "stash.appscode.com"
	VersionTag = StashKey + "/tag"

	KeyDeleteJobOnCompletion     = StashKey + "/delete-job-on-completion"
	AllowDeletingJobOnCompletion = "true"
)

const (
	KindDeployment            = "Deployment"
	KindReplicaSet            = "ReplicaSet"
	KindReplicationController = "ReplicationController"
	KindStatefulSet           = "StatefulSet"
	KindDaemonSet             = "DaemonSet"
	KindPod                   = "Pod"
	KindPersistentVolumeClaim = "PersistentVolumeClaim"
	KindAppBinding            = "AppBinding"
	KindDeploymentConfig      = "DeploymentConfig"
	KindSecret                = "Secret"
	KindService               = "Service"
	KindJob                   = "Job"
	KindCronJob               = "CronJob"
)

const (
	ResourcePluralDeployment            = "deployments"
	ResourcePluralReplicaSet            = "replicasets"
	ResourcePluralReplicationController = "replicationcontrollers"
	ResourcePluralStatefulSet           = "statefulsets"
	ResourcePluralDaemonSet             = "daemonsets"
	ResourcePluralPod                   = "pods"
	ResourcePluralPersistentVolumeClaim = "persistentvolumeclaims"
	ResourcePluralAppBinding            = "appbindings"
	ResourcePluralDeploymentConfig      = "deploymentconfigs"
	ResourcePluralSecret                = "secrets"
	ResourcePluralService               = "services"
)
