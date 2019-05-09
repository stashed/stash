package v1beta1

import (
	"hash/fnv"
	"strconv"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
	crdutils "kmodules.xyz/client-go/apiextensions/v1beta1"
)

func (btpl BackupConfigurationTemplate) GetSpecHash() string {
	hash := fnv.New64a()
	hashutil.DeepHashObject(hash, btpl.Spec)
	return strconv.FormatUint(hash.Sum64(), 10)
}

func (btpl BackupConfigurationTemplate) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crdutils.NewCustomResourceDefinition(crdutils.Config{
		Group:         SchemeGroupVersion.Group,
		Plural:        ResourcePluralBackupConfigurationTemplate,
		Singular:      ResourceSingularBackupConfigurationTemplate,
		Kind:          ResourceKindBackupConfigurationTemplate,
		ShortNames:    []string{"btpl"},
		Categories:    []string{"stash", "appscode", "backup"},
		ResourceScope: string(apiextensions.ClusterScoped),
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
		SpecDefinitionName:    "stash.appscode.dev/stash/apis/stash/v1beta1.BackupConfigurationTemplate",
		EnableValidation:      true,
		GetOpenAPIDefinitions: GetOpenAPIDefinitions,
		AdditionalPrinterColumns: []apiextensions.CustomResourceColumnDefinition{
			{
				Name:     "Task",
				Type:     "string",
				JSONPath: ".spec.task.name",
			},
			{
				Name:     "Schedule",
				Type:     "string",
				JSONPath: ".spec.schedule",
			},
			{
				Name:     "Age",
				Type:     "date",
				JSONPath: ".metadata.creationTimestamp",
			},
		},
	})
}
