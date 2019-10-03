package v1beta1

import (
	"hash/fnv"
	"strconv"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
	crdutils "kmodules.xyz/client-go/apiextensions/v1beta1"
	meta_util "kmodules.xyz/client-go/meta"
)

func (b BackupConfiguration) GetSpecHash() string {
	hash := fnv.New64a()
	hashutil.DeepHashObject(hash, b.Spec)
	return strconv.FormatUint(hash.Sum64(), 10)
}

func (b BackupConfiguration) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crdutils.NewCustomResourceDefinition(crdutils.Config{
		Group:         SchemeGroupVersion.Group,
		Plural:        ResourcePluralBackupConfiguration,
		Singular:      ResourceSingularBackupConfiguration,
		Kind:          ResourceKindBackupConfiguration,
		ShortNames:    []string{"bc"},
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
		SpecDefinitionName:    "stash.appscode.dev/stash/apis/stash/v1beta1.BackupConfiguration",
		EnableValidation:      true,
		GetOpenAPIDefinitions: GetOpenAPIDefinitionsWithRetentionPolicy,
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
