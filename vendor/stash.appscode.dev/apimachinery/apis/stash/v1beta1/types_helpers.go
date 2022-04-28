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

package v1beta1

import (
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"

	"k8s.io/kube-openapi/pkg/common"
)

const (
	StashBackupComponent  = "stash-backup"
	StashRestoreComponent = "stash-restore"
	TargetKindEmpty       = "EmptyTarget"
)

// TODO: complete
func (t TargetRef) IsWorkload() bool {
	return t.Kind == "Deployment"
}

func GetOpenAPIDefinitionsWithRetentionPolicy(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	key := "stash.appscode.dev/apimachinery/apis/stash/v1alpha1.RetentionPolicy"
	out := GetOpenAPIDefinitions(ref)
	out[key] = v1alpha1.GetOpenAPIDefinitions(ref)[key]
	return out
}

func EmptyTargetRef() TargetRef {
	return TargetRef{
		APIVersion: "na",
		Kind:       TargetKindEmpty,
		Name:       "na",
	}
}
