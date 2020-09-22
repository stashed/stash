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

// This file contains consts that are not shared between components and set just internally.
// They will likely be removed in (near) future.

const (
	// DeployerPodCreatedAtAnnotation is an annotation on a deployment that
	// records the time in RFC3339 format of when the deployer pod for this particular
	// deployment was created.
	// This is set by deployer controller, but not consumed by any command or internally.
	// DEPRECATED: will be removed soon
	DeployerPodCreatedAtAnnotation = "openshift.io/deployer-pod.created-at"

	// DeployerPodStartedAtAnnotation is an annotation on a deployment that
	// records the time in RFC3339 format of when the deployer pod for this particular
	// deployment was started.
	// This is set by deployer controller, but not consumed by any command or internally.
	// DEPRECATED: will be removed soon
	DeployerPodStartedAtAnnotation = "openshift.io/deployer-pod.started-at"

	// DeployerPodCompletedAtAnnotation is an annotation on deployment that records
	// the time in RFC3339 format of when the deployer pod finished.
	// This is set by deployer controller, but not consumed by any command or internally.
	// DEPRECATED: will be removed soon
	DeployerPodCompletedAtAnnotation = "openshift.io/deployer-pod.completed-at"

	// DesiredReplicasAnnotation represents the desired number of replicas for a
	// new deployment.
	// This is set by deployer controller, but not consumed by any command or internally.
	// DEPRECATED: will be removed soon
	DesiredReplicasAnnotation = "kubectl.kubernetes.io/desired-replicas"

	// DeploymentAnnotation is an annotation on a deployer Pod. The annotation value is the name
	// of the deployment (a ReplicationController) on which the deployer Pod acts.
	// This is set by deployer controller and consumed internally and in oc adm top command.
	// DEPRECATED: will be removed soon
	DeploymentAnnotation = "openshift.io/deployment.name"
)
