package v1beta1

import (
	"k8s.io/kube-openapi/pkg/common"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
)

const (
	StashBackupComponent  = "stash-backup"
	StashRestoreComponent = "stash-restore"
)

// TODO: complete
func (t TargetRef) IsWorkload() bool {
	if t.Kind == "Deployment" {
		return true
	}
	return false
}

func GetOpenAPIDefinitionsWithRetentionPolicy(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	key := "stash.appscode.dev/stash/apis/stash/v1alpha1.RetentionPolicy"
	out := GetOpenAPIDefinitions(ref)
	out[key] = v1alpha1.GetOpenAPIDefinitions(ref)[key]
	return out
}
