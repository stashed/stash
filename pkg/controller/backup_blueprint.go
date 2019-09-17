package controller

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"kmodules.xyz/webhook-runtime/admission"
	hooks "kmodules.xyz/webhook-runtime/admission/v1beta1"
	webhook "kmodules.xyz/webhook-runtime/admission/v1beta1/generic"
	"stash.appscode.dev/stash/apis/stash"
	api_v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
)

func (c *StashController) NewBackupBlueprintWebhook() hooks.AdmissionHook {
	return webhook.NewGenericWebhook(
		schema.GroupVersionResource{
			Group:    "admission.stash.appscode.com",
			Version:  "v1beta1",
			Resource: "backupblueprintvalidators",
		},
		"backupblueprintvalidator",
		[]string{stash.GroupName},
		api_v1beta1.SchemeGroupVersion.WithKind(api_v1beta1.ResourceKindBackupBlueprint),
		nil,
		&admission.ResourceHandlerFuncs{
			CreateFunc: func(obj runtime.Object) (runtime.Object, error) {
				return nil, obj.(*api_v1beta1.BackupBlueprint).IsValid()
			},
		},
	)
}
