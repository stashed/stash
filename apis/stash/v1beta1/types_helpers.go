package v1beta1

import (
	"stash.appscode.dev/stash/apis/stash/v1alpha1"

	"k8s.io/kube-openapi/pkg/common"
)

const (
	StashBackupComponent  = "stash-backup"
	StashRestoreComponent = "stash-restore"
)

// TODO: complete
func (t TargetRef) IsWorkload() bool {
	return t.Kind == "Deployment"
}

func GetOpenAPIDefinitionsWithRetentionPolicy(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	key := "stash.appscode.dev/stash/apis/stash/v1alpha1.RetentionPolicy"
	out := GetOpenAPIDefinitions(ref)
	out[key] = v1alpha1.GetOpenAPIDefinitions(ref)[key]
	return out
}
