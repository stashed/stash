/*
Copyright AppsCode Inc. and Contributors

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

package util

import (
	"context"
	"encoding/json"

	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	"stash.appscode.dev/apimachinery/pkg/invoker"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appcat "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	appcatalog_cs "kmodules.xyz/custom-resources/client/clientset/versioned"
)

func ExtractAddonInfo(appClient appcatalog_cs.Interface, task v1beta1.TaskRef, targetRef v1beta1.TargetRef) (*appcat.StashTaskSpec, error) {
	var params appcat.StashAddon

	// If the target is AppBinding and it has addon information set in the parameters section, then extract the addon info.
	if invoker.TargetOfGroupKind(targetRef, appcat.SchemeGroupVersion.Group, appcat.ResourceKindApp) {
		// get the AppBinding
		appBinding, err := appClient.AppcatalogV1alpha1().AppBindings(targetRef.Namespace).Get(context.TODO(), targetRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		// extract the parameters
		if appBinding.Spec.Parameters != nil {
			err = json.Unmarshal(appBinding.Spec.Parameters.Raw, &params)
			if err != nil {
				return nil, err
			}
		}
	}

	addon := params.Stash.Addon

	// If the user provides Task information in the backup/restore invoker spec, it should have higher precedence.
	// We don't know whether this function was called from BackupSession controller or RestoreSession controller.
	// Hence, we are going to overwrite the task name & parameters in both backupTask & restoreTask section.
	// It does not have any adverse effect because when it is called from the BackupSession controller, we will overwrite with backup task info
	// and when it is called from the RestoreSession controller, we will overwrite with restore task info.
	if task.Name != "" {
		addon.BackupTask.Name = task.Name
		addon.RestoreTask.Name = task.Name
	}
	if len(task.Params) != 0 {
		addon.BackupTask.Params = getTaskParams(task)
		addon.RestoreTask.Params = getTaskParams(task)
	}

	return &addon, nil
}

func getTaskParams(task v1beta1.TaskRef) []appcat.Param {
	params := make([]appcat.Param, len(task.Params))
	for i := range task.Params {
		params[i].Name = task.Params[i].Name
		params[i].Value = task.Params[i].Value
	}
	return params
}
