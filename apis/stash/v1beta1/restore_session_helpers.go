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

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
	crdutils "kmodules.xyz/client-go/apiextensions/v1beta1"
	meta_util "kmodules.xyz/client-go/meta"
)

func (r RestoreSession) GetSpecHash() string {
	hash := fnv.New64a()
	hashutil.DeepHashObject(hash, r.Spec)
	return strconv.FormatUint(hash.Sum64(), 10)
}

func (r RestoreSession) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crdutils.NewCustomResourceDefinition(crdutils.Config{
		Group:         SchemeGroupVersion.Group,
		Plural:        ResourcePluralRestoreSession,
		Singular:      ResourceSingularRestoreSession,
		Kind:          ResourceKindRestoreSession,
		ShortNames:    []string{"restore"},
		Categories:    []string{"stash", "appscode", "all"},
		ResourceScope: string(apiextensions.NamespaceScoped),
		Versions: []apiextensions.CustomResourceDefinitionVersion{
			{
				Name:    SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
			},
		},
		Labels: crdutils.Labels{
			LabelsMap: map[string]string{"app": "stash"},
		},
		SpecDefinitionName:      "stash.appscode.dev/stash/apis/stash/v1beta1.RestoreSession",
		EnableValidation:        true,
		GetOpenAPIDefinitions:   GetOpenAPIDefinitions,
		EnableStatusSubresource: true,
		AdditionalPrinterColumns: []apiextensions.CustomResourceColumnDefinition{
			{
				Name:     "Repository",
				Type:     "string",
				JSONPath: ".spec.repository.name",
			},
			{
				Name:     "Phase",
				Type:     "string",
				JSONPath: ".status.phase",
			},
			{
				Name:     "Age",
				Type:     "date",
				JSONPath: ".metadata.creationTimestamp",
			},
		},
	})
}

// OffshootLabels return labels consist of the labels provided by user to BackupConfiguration crd and
// stash specific generic labels. It overwrites the the user provided labels if it matched with stash specific generic labels.
func (r RestoreSession) OffshootLabels() map[string]string {
	overrides := make(map[string]string)
	overrides[meta_util.ComponentLabelKey] = StashRestoreComponent
	overrides[meta_util.ManagedByLabelKey] = StashKey

	return upsertLabels(r.Labels, overrides)
}
