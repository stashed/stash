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

func (RestoreSession) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crds.MustCustomResourceDefinition(SchemeGroupVersion.WithResource(ResourcePluralRestoreSession))
}

func (r RestoreSession) GetSpecHash() string {
	hash := fnv.New64a()
	meta_util.DeepHashObject(hash, r.Spec)
	return strconv.FormatUint(hash.Sum64(), 10)
}

// OffshootLabels return labels consist of the labels provided by user to BackupConfiguration crd and
// stash specific generic labels. It overwrites the the user provided labels if it matched with stash specific generic labels.
func (r *RestoreSession) OffshootLabels() map[string]string {
	overrides := make(map[string]string)
	overrides[meta_util.ComponentLabelKey] = StashRestoreComponent
	overrides[meta_util.ManagedByLabelKey] = StashKey

	return upsertLabels(r.Labels, overrides)
}

// Migrate moved deprecated fields into the appropriate fields
func (r *RestoreSession) Migrate() {
	// move the deprecated "rules" section of ".Spec" section, into the "rules" section under ".Spec.BackupTargetStatus" section
	if len(r.Spec.Rules) > 0 && r.Spec.Target != nil {
		r.Spec.Target.Rules = r.Spec.Rules
		r.Spec.Rules = nil
	}
}
