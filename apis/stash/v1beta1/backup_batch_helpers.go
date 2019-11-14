/*
Copyright The Stash Authors.

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

	"stash.appscode.dev/stash/api/crds"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
	meta_util "kmodules.xyz/client-go/meta"
	"sigs.k8s.io/yaml"
)

func (_ BackupBatch) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	data := crds.MustAsset("stash.appscode.com_backupbatches.yaml")
	var out apiextensions.CustomResourceDefinition
	utilruntime.Must(yaml.Unmarshal(data, &out))
	return &out
}

func (b BackupBatch) GetSpecHash() string {
	hash := fnv.New64a()
	hashutil.DeepHashObject(hash, b.Spec)
	return strconv.FormatUint(hash.Sum64(), 10)
}

// OffshootLabels return labels consist of the labels provided by user to BackupBatch crd and
// stash specific generic labels. It overwrites the the user provided labels if it matched with stash specific generic labels.
func (b BackupBatch) OffshootLabels() map[string]string {
	overrides := make(map[string]string)
	overrides[meta_util.ComponentLabelKey] = StashBackupComponent
	overrides[meta_util.ManagedByLabelKey] = StashKey

	return upsertLabels(b.Labels, overrides)
}
