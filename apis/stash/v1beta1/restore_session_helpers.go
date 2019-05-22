package v1beta1

import (
	"hash/fnv"
	"strconv"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
	crdutils "kmodules.xyz/client-go/apiextensions/v1beta1"
	meta_util "kmodules.xyz/client-go/meta"
	"stash.appscode.dev/stash/apis"
)

func (r RestoreSession) GetSpecHash() string {
	hash := fnv.New64a()
	hashutil.DeepHashObject(hash, r.Spec)
	return strconv.FormatUint(hash.Sum64(), 10)
}

func (c RestoreSession) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crdutils.NewCustomResourceDefinition(crdutils.Config{
		Group:         SchemeGroupVersion.Group,
		Plural:        ResourcePluralRestoreSession,
		Singular:      ResourceSingularRestoreSession,
		Kind:          ResourceKindRestoreSession,
		ShortNames:    []string{"restore"},
		Categories:    []string{"stash", "appscode", "restore"},
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
		EnableStatusSubresource: apis.EnableStatusSubresource,
		AdditionalPrinterColumns: []apiextensions.CustomResourceColumnDefinition{
			{
				Name:     "Repository-Name",
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
func (rs RestoreSession) OffshootLabels(name string) map[string]string {
	genericLabels := make(map[string]string, 0)
	genericLabels[meta_util.NameLabelKey] = name
	genericLabels[meta_util.InstanceLabelKey] = rs.Name
	genericLabels[meta_util.ComponentLabelKey] = StashBackupComponent
	genericLabels[meta_util.ManagedByLabelKey] = StashKey
	return meta_util.FilterKeys(StashKey, genericLabels, rs.Labels)
}
