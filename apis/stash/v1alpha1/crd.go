package v1alpha1

import (
	crdutils "github.com/appscode/kutil/apiextensions/v1beta1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

var (
	EnableStatusSubresource bool
)

func (c Restic) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crdutils.NewCustomResourceDefinition(crdutils.Config{
		Group:         SchemeGroupVersion.Group,
		Plural:        ResourcePluralRestic,
		Singular:      ResourceSingularRestic,
		Kind:          ResourceKindRestic,
		ShortNames:    []string{"rst"},
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
		SpecDefinitionName:      "github.com/appscode/stash/apis/stash/v1alpha1.Restic",
		EnableValidation:        true,
		GetOpenAPIDefinitions:   GetOpenAPIDefinitions,
		EnableStatusSubresource: EnableStatusSubresource,
		AdditionalPrinterColumns: []apiextensions.CustomResourceColumnDefinition{
			{
				Name:     "Selector",
				Type:     "string",
				JSONPath: ".spec.selector",
			},
			{
				Name:     "Schedule",
				Type:     "string",
				JSONPath: ".spec.schedule",
			},
			{
				Name:     "Backup-Type",
				Type:     "string",
				JSONPath: ".spec.type",
				Priority: 10,
			},
			{
				Name:     "Paused",
				Type:     "boolean",
				JSONPath: ".spec.paused",
			},
			{
				Name:     "Age",
				Type:     "date",
				JSONPath: ".metadata.creationTimestamp",
			},
		},
	})
}

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
		SpecDefinitionName:      "github.com/appscode/stash/apis/stash/v1alpha1.Recovery",
		EnableValidation:        true,
		GetOpenAPIDefinitions:   GetOpenAPIDefinitions,
		EnableStatusSubresource: EnableStatusSubresource,
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

func (c Repository) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crdutils.NewCustomResourceDefinition(crdutils.Config{
		Group:         SchemeGroupVersion.Group,
		Plural:        ResourcePluralRepository,
		Singular:      ResourceSingularRepository,
		Kind:          ResourceKindRepository,
		ShortNames:    []string{"repo"},
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
		SpecDefinitionName:      "github.com/appscode/stash/apis/stash/v1alpha1.Repository",
		EnableValidation:        true,
		GetOpenAPIDefinitions:   GetOpenAPIDefinitions,
		EnableStatusSubresource: EnableStatusSubresource,
		AdditionalPrinterColumns: []apiextensions.CustomResourceColumnDefinition{
			{
				Name:     "Backup-Count",
				Type:     "integer",
				JSONPath: ".status.backupCount",
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
