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
	"os"
	"strconv"

	core "k8s.io/api/core/v1"
)

type RuntimeSettings struct {
	Pod       *PodRuntimeSettings       `json:"pod,omitempty" protobuf:"bytes,1,opt,name=pod"`
	Container *ContainerRuntimeSettings `json:"container,omitempty" protobuf:"bytes,2,opt,name=container"`
}

type PodRuntimeSettings struct {
	// PodAnnotations are the annotations that will be attached with the respective Pod
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty" protobuf:"bytes,15,rep,name=podAnnotations"`
	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty" protobuf:"bytes,1,rep,name=nodeSelector"`
	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty" protobuf:"bytes,2,opt,name=serviceAccountName"`
	// ServiceAccountAnnotations are the annotations that will be attached with the respective ServiceAccount
	// +optional
	ServiceAccountAnnotations map[string]string `json:"serviceAccountAnnotations" protobuf:"bytes,16,rep,name=serviceAccountAnnotations"`
	// AutomountServiceAccountToken indicates whether a service account token should be automatically mounted.
	// +optional
	AutomountServiceAccountToken *bool `json:"automountServiceAccountToken,omitempty" protobuf:"varint,3,opt,name=automountServiceAccountToken"`
	// NodeName is a request to schedule this pod onto a specific node. If it is non-empty,
	// the scheduler simply schedules this pod onto that node, assuming that it fits resource
	// requirements.
	// +optional
	NodeName string `json:"nodeName,omitempty" protobuf:"bytes,4,opt,name=nodeName"`
	// Security options the pod should run with.
	// More info: https://kubernetes.io/docs/concepts/policy/security-context/
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/
	// +optional
	SecurityContext *core.PodSecurityContext `json:"securityContext,omitempty" protobuf:"bytes,5,opt,name=securityContext"`
	// ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodRuntimeSettings.
	// If specified, these secrets will be passed to individual puller implementations for them to use. For example,
	// in the case of docker, only DockerConfig type secrets are honored.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	ImagePullSecrets []core.LocalObjectReference `json:"imagePullSecrets,omitempty" protobuf:"bytes,6,rep,name=imagePullSecrets"`
	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *core.Affinity `json:"affinity,omitempty" protobuf:"bytes,7,opt,name=affinity"`
	// If specified, the pod will be dispatched by specified scheduler.
	// If not specified, the pod will be dispatched by default scheduler.
	// +optional
	SchedulerName string `json:"schedulerName,omitempty" protobuf:"bytes,8,opt,name=schedulerName"`
	// If specified, the pod's tolerations.
	// +optional
	Tolerations []core.Toleration `json:"tolerations,omitempty" protobuf:"bytes,9,rep,name=tolerations"`
	// If specified, indicates the pod's priority. "system-node-critical" and
	// "system-cluster-critical" are two special keywords which indicate the
	// highest priorities with the former being the highest priority. Any other
	// name must be defined by creating a PriorityClass object with that name.
	// If not specified, the pod priority will be default or zero if there is no
	// default.
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty" protobuf:"bytes,10,opt,name=priorityClassName"`
	// The priority value. Various system components use this field to find the
	// priority of the pod. When Priority Admission Controller is enabled, it
	// prevents users from setting this field. The admission controller populates
	// this field from PriorityClassName.
	// The higher the value, the higher the priority.
	// +optional
	Priority *int32 `json:"priority,omitempty" protobuf:"varint,11,opt,name=priority"`
	// If specified, all readiness gates will be evaluated for pod readiness.
	// A pod is ready when all its containers are ready AND
	// all conditions specified in the readiness gates have status equal to "True"
	// More info: https://git.k8s.io/enhancements/keps/sig-network/0007-pod-ready%2B%2B.md
	// +optional
	ReadinessGates []core.PodReadinessGate `json:"readinessGates,omitempty" protobuf:"bytes,12,rep,name=readinessGates"`
	// RuntimeClassName refers to a RuntimeClass object in the node.k8s.io group, which should be used
	// to run this pod.  If no RuntimeClass resource matches the named class, the pod will not be run.
	// If unset or empty, the "legacy" RuntimeClass will be used, which is an implicit class with an
	// empty definition that uses the default runtime handler.
	// More info: https://git.k8s.io/enhancements/keps/sig-node/runtime-class.md
	// This is an alpha feature and may change in the future.
	// +optional
	RuntimeClassName *string `json:"runtimeClassName,omitempty" protobuf:"bytes,13,opt,name=runtimeClassName"`
	// EnableServiceLinks indicates whether information about services should be injected into pod's
	// environment variables, matching the syntax of Docker links.
	// Optional: Defaults to true.
	// +optional
	EnableServiceLinks *bool `json:"enableServiceLinks,omitempty" protobuf:"varint,14,opt,name=enableServiceLinks"`
}

type ContainerRuntimeSettings struct {
	// Compute Resources required by container.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	// +optional
	Resources core.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,1,opt,name=resources"`
	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	LivenessProbe *core.Probe `json:"livenessProbe,omitempty" protobuf:"bytes,2,opt,name=livenessProbe"`
	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes
	// +optional
	ReadinessProbe *core.Probe `json:"readinessProbe,omitempty" protobuf:"bytes,3,opt,name=readinessProbe"`
	// Actions that the management system should take in response to container lifecycle events.
	// Cannot be updated.
	// +optional
	Lifecycle *core.Lifecycle `json:"lifecycle,omitempty" protobuf:"bytes,4,opt,name=lifecycle"`
	// Security options the pod should run with.
	// More info: https://kubernetes.io/docs/concepts/policy/security-context/
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/
	// +optional
	SecurityContext *core.SecurityContext `json:"securityContext,omitempty" protobuf:"bytes,5,opt,name=securityContext"`
	// Settings to configure `nice` to throttle the load on cpu.
	// More info: http://kennystechtalk.blogspot.com/2015/04/throttling-cpu-usage-with-linux-cgroups.html
	// More info: https://oakbytes.wordpress.com/2012/06/06/linux-scheduler-cfs-and-nice/
	// +optional
	Nice *NiceSettings `json:"nice,omitempty" protobuf:"bytes,6,opt,name=nice"`
	// Settings to configure `ionice` to throttle the load on disk.
	// More info: http://kennystechtalk.blogspot.com/2015/04/throttling-cpu-usage-with-linux-cgroups.html
	// More info: https://oakbytes.wordpress.com/2012/06/06/linux-scheduler-cfs-and-nice/
	// +optional
	IONice *IONiceSettings `json:"ionice,omitempty" protobuf:"bytes,7,opt,name=ionice"`
	// List of sources to populate environment variables in the container.
	// The keys defined within a source must be a C_IDENTIFIER. All invalid keys
	// will be reported as an event when the container is starting. When a key exists in multiple
	// sources, the value associated with the last source will take precedence.
	// Values defined by an Env with a duplicate key will take precedence.
	// Cannot be updated.
	// +optional
	EnvFrom []core.EnvFromSource `json:"envFrom,omitempty" protobuf:"bytes,8,rep,name=envFrom"`
	// List of environment variables to set in the container.
	// Cannot be updated.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	Env []core.EnvVar `json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,9,rep,name=env"`
}

// https://linux.die.net/man/1/nice
type NiceSettings struct {
	Adjustment *int32 `json:"adjustment,omitempty" protobuf:"varint,1,opt,name=adjustment"`
}

// https://linux.die.net/man/1/ionice
type IONiceSettings struct {
	Class     *int32 `json:"class,omitempty" protobuf:"varint,1,opt,name=class"`
	ClassData *int32 `json:"classData,omitempty" protobuf:"varint,2,opt,name=classData"`
}

func NiceSettingsFromEnv() (*NiceSettings, error) {
	var settings *NiceSettings
	if v, ok := os.LookupEnv(NiceAdjustment); ok {
		vi, err := parseInt32P(v)
		if err != nil {
			return nil, err
		}
		settings = &NiceSettings{
			Adjustment: vi,
		}
	}
	return settings, nil
}

func IONiceSettingsFromEnv() (*IONiceSettings, error) {
	var settings *IONiceSettings
	if v, ok := os.LookupEnv(IONiceClass); ok {
		vi, err := parseInt32P(v)
		if err != nil {
			return nil, err
		}
		settings = &IONiceSettings{
			Class: vi,
		}
	}
	if v, ok := os.LookupEnv(IONiceClassData); ok {
		vi, err := parseInt32P(v)
		if err != nil {
			return nil, err
		}
		if settings == nil {
			settings = &IONiceSettings{}
		}
		settings.ClassData = vi
	}
	return settings, nil
}

func parseInt32P(v string) (*int32, error) {
	vi, err := strconv.Atoi(v)
	if err != nil {
		return nil, err
	}
	out := int32(vi)
	return &out, nil
}
