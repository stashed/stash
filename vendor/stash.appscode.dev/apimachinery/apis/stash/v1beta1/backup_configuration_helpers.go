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
	"hash/fnv"
	"strconv"

	"stash.appscode.dev/apimachinery/crds"

	"kmodules.xyz/client-go/apiextensions"
	meta_util "kmodules.xyz/client-go/meta"
)

func (BackupConfiguration) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crds.MustCustomResourceDefinition(SchemeGroupVersion.WithResource(ResourcePluralBackupConfiguration))
}

func (b BackupConfiguration) GetSpecHash() string {
	hash := fnv.New64a()
	meta_util.DeepHashObject(hash, b.Spec)
	return strconv.FormatUint(hash.Sum64(), 10)
}

// OffshootLabels return labels consist of the labels provided by user to BackupConfiguration crd and
// stash specific generic labels. It overwrites the the user provided labels if it matched with stash specific generic labels.
func (b BackupConfiguration) OffshootLabels() map[string]string {
	overrides := make(map[string]string)
	overrides[meta_util.ComponentLabelKey] = StashBackupComponent
	overrides[meta_util.ManagedByLabelKey] = StashKey

	return upsertLabels(b.Labels, overrides)
}

func upsertLabels(originalLabels, overrides map[string]string) map[string]string {
	if originalLabels == nil {
		originalLabels = make(map[string]string, len(overrides))
	}
	for k, v := range overrides {
		originalLabels[k] = v
	}
	return originalLabels
}
