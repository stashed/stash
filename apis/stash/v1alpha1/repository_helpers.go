package v1alpha1

import (
	crdutils "github.com/appscode/kutil/apiextensions/v1beta1"
	"github.com/appscode/stash/apis"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

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
		EnableStatusSubresource: apis.EnableStatusSubresource,
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
