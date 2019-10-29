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
package v1alpha1

import (
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	crdutils "kmodules.xyz/client-go/apiextensions/v1beta1"
)

func (c Repository) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crdutils.NewCustomResourceDefinition(crdutils.Config{
		Group:         SchemeGroupVersion.Group,
		Plural:        ResourcePluralRepository,
		Singular:      ResourceSingularRepository,
		Kind:          ResourceKindRepository,
		ShortNames:    []string{"repo"},
		Categories:    []string{"stash", "appscode"},
		ResourceScope: string(apiextensions.NamespaceScoped),
		Versions: []apiextensions.CustomResourceDefinitionVersion{
			{
				Name:    SchemeGroupVersion.Version,
				Served:  true,
				Storage: true,
			},
			{
				Name:    "v1beta1",
				Served:  true,
				Storage: false,
			},
		},
		Labels: crdutils.Labels{
			LabelsMap: map[string]string{"app": "stash"},
		},
		SpecDefinitionName:      "stash.appscode.dev/stash/apis/stash/v1beta1.Repository",
		EnableValidation:        true,
		GetOpenAPIDefinitions:   GetOpenAPIDefinitions,
		EnableStatusSubresource: true,
		AdditionalPrinterColumns: []apiextensions.CustomResourceColumnDefinition{
			{
				Name:     "Integrity",
				Type:     "boolean",
				JSONPath: ".status.integrity",
			},
			{
				Name:     "Size",
				Type:     "string",
				JSONPath: ".status.size",
			},
			{
				Name:     "Snapshot-Count",
				Type:     "integer",
				JSONPath: ".status.snapshotCount",
			},
			{
				Name:     "Last-Successful-Backup",
				Type:     "date",
				JSONPath: ".status.lastBackupTime",
				Format:   "date-time",
			},
			{
				Name:     "Age",
				Type:     "date",
				JSONPath: ".metadata.creationTimestamp",
			},
		},
	})
}
