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

import "strings"

// ResourceShortForm takes a resource kind and returns the short form of the resource.
// xref: https://kubernetes.io/docs/reference/kubectl/overview/#resource-types
func ResourceShortForm(kind string) string {
	switch kind {
	case KindDeployment:
		return "deploy"
	case KindReplicationController:
		return "rc"
	case KindDaemonSet:
		return "ds"
	case KindStatefulSet:
		return "sts"
	case KindPersistentVolumeClaim:
		return "pvc"
	case KindPod:
		return "po"
	case KindAppBinding:
		return "app"
	default:
		return strings.ToLower(kind)
	}
}
