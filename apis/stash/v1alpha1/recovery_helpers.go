package v1alpha1

import (
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	crdutils "kmodules.xyz/client-go/apiextensions/v1beta1"
	"stash.appscode.dev/stash/apis"
)

func (c Recovery) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crdutils.NewCustomResourceDefinition(crdutils.Config{
		Group:         SchemeGroupVersion.Group,
		Plural:        ResourcePluralRecovery,
		Singular:      ResourceSingularRecovery,
		Kind:          ResourceKindRecovery,
		ShortNames:    []string{"rec"},
		Categories:    []string{"storage", "appscode", "all"},
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
		SpecDefinitionName:      "stash.appscode.dev/stash/apis/stash/v1alpha1.Recovery",
		EnableValidation:        true,
		GetOpenAPIDefinitions:   GetOpenAPIDefinitions,
		EnableStatusSubresource: apis.EnableStatusSubresource,
		AdditionalPrinterColumns: []apiextensions.CustomResourceColumnDefinition{
			{
				Name:     "Repository-Namespace",
				Type:     "string",
				JSONPath: ".spec.repository.namespace",
			},
			{
				Name:     "Repository-Name",
				Type:     "string",
				JSONPath: ".spec.repository.name",
			},
			{
				Name:     "Snapshot",
				Type:     "string",
				JSONPath: ".spec.snapshot",
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
