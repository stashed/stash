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
	core_util "kmodules.xyz/client-go/core/v1"
	ofstv2 "kmodules.xyz/offshoot-api/api/v2"

	core "k8s.io/api/core/v1"
)

// EnsureContainerExists ensures that given container either exits by default or
// it will create the container, then insert it to the podTemplate and return a pointer of that container
func EnsureContainerExists(podTemplate *ofstv2.PodTemplateSpec, containerName string) *core.Container {
	container := core_util.GetContainerByName(podTemplate.Spec.Containers, containerName)
	if container == nil {
		container = &core.Container{
			Name: containerName,
		}
	}
	podTemplate.Spec.Containers = core_util.UpsertContainer(podTemplate.Spec.Containers, *container)
	return core_util.GetContainerByName(podTemplate.Spec.Containers, containerName)
}

// EnsureInitContainerExists ensures that given initContainer either exits by default or
// it will create the initContainer, then insert it to the podTemplate and return a pointer of that initContainer
func EnsureInitContainerExists(podTemplate *ofstv2.PodTemplateSpec, containerName string) *core.Container {
	container := core_util.GetContainerByName(podTemplate.Spec.InitContainers, containerName)
	if container == nil {
		container = &core.Container{
			Name: containerName,
		}
	}
	podTemplate.Spec.InitContainers = core_util.UpsertContainer(podTemplate.Spec.InitContainers, *container)
	return core_util.GetContainerByName(podTemplate.Spec.InitContainers, containerName)
}
