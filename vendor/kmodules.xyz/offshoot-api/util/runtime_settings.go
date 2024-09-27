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

package util

import (
	"fmt"

	core_util "kmodules.xyz/client-go/core/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"

	core "k8s.io/api/core/v1"
)

const (
	NiceAdjustment  = "NICE_ADJUSTMENT"
	IONiceClass     = "IONICE_CLASS"
	IONiceClassData = "IONICE_CLASS_DATA"
)

func ApplyContainerRuntimeSettings(container core.Container, settings ofst.ContainerRuntimeSettings) core.Container {
	if len(settings.Resources.Limits) > 0 {
		container.Resources.Limits = settings.Resources.Limits
	}
	if len(settings.Resources.Limits) > 0 {
		container.Resources.Requests = settings.Resources.Requests
	}
	if settings.LivenessProbe != nil {
		container.LivenessProbe = settings.LivenessProbe
	}
	if settings.ReadinessProbe != nil {
		container.ReadinessProbe = settings.ReadinessProbe
	}
	if settings.Lifecycle != nil {
		container.Lifecycle = settings.Lifecycle
	}
	if settings.SecurityContext != nil {
		container.SecurityContext = settings.SecurityContext
	}
	if len(settings.EnvFrom) > 0 {
		container.EnvFrom = append(container.EnvFrom, settings.EnvFrom...)
	}
	if len(settings.Env) > 0 {
		container.Env = core_util.UpsertEnvVars(container.Env, settings.Env...)
	}
	// set nice, ionice settings as env
	if settings.Nice != nil && settings.Nice.Adjustment != nil {
		container.Env = core_util.UpsertEnvVars(container.Env, core.EnvVar{
			Name:  NiceAdjustment,
			Value: fmt.Sprint(*settings.Nice.Adjustment),
		})
	}
	if settings.IONice != nil {
		if settings.IONice.Class != nil {
			container.Env = core_util.UpsertEnvVars(container.Env, core.EnvVar{
				Name:  IONiceClass,
				Value: fmt.Sprint(*settings.IONice.Class),
			})
		}
		if settings.IONice.ClassData != nil {
			container.Env = core_util.UpsertEnvVars(container.Env, core.EnvVar{
				Name:  IONiceClassData,
				Value: fmt.Sprint(*settings.IONice.ClassData),
			})
		}
	}
	return container
}

func ApplyPodRuntimeSettings(podSpec core.PodSpec, settings ofst.PodRuntimeSettings) core.PodSpec {
	if len(settings.NodeSelector) > 0 {
		podSpec.NodeSelector = settings.NodeSelector
	}
	if settings.ServiceAccountName != "" {
		podSpec.ServiceAccountName = settings.ServiceAccountName
	}
	if settings.AutomountServiceAccountToken != nil {
		podSpec.AutomountServiceAccountToken = settings.AutomountServiceAccountToken
	}
	if settings.NodeName != "" {
		podSpec.NodeName = settings.NodeName
	}
	if settings.SecurityContext != nil {
		podSpec.SecurityContext = settings.SecurityContext
	}
	if len(settings.ImagePullSecrets) > 0 {
		podSpec.ImagePullSecrets = core_util.MergeLocalObjectReferences(podSpec.ImagePullSecrets, settings.ImagePullSecrets)
	}
	if settings.Affinity != nil {
		podSpec.Affinity = settings.Affinity
	}
	if settings.SchedulerName != "" {
		podSpec.SchedulerName = settings.SchedulerName
	}
	if len(settings.Tolerations) > 0 {
		podSpec.Tolerations = settings.Tolerations
	}
	if settings.PriorityClassName != "" {
		podSpec.PriorityClassName = settings.PriorityClassName
	}
	if settings.Priority != nil {
		podSpec.Priority = settings.Priority
	}
	if len(settings.ReadinessGates) > 0 {
		podSpec.ReadinessGates = settings.ReadinessGates
	}
	if settings.RuntimeClassName != nil {
		podSpec.RuntimeClassName = settings.RuntimeClassName
	}
	if settings.EnableServiceLinks != nil {
		podSpec.EnableServiceLinks = settings.EnableServiceLinks
	}
	if settings.TopologySpreadConstraints != nil {
		podSpec.TopologySpreadConstraints = settings.TopologySpreadConstraints
	}
	return podSpec
}
