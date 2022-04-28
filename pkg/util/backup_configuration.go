/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"bytes"
	"fmt"
	"reflect"

	v1beta1_api "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	v1beta1_listers "stash.appscode.dev/apimachinery/client/listers/stash/v1beta1"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	"kmodules.xyz/client-go/meta"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

// GetAppliedBackupConfiguration check whether BackupConfiguration was applied as annotation and returns the object definition if exist.
func GetAppliedBackupConfiguration(m map[string]string) (*v1beta1_api.BackupConfiguration, error) {
	data := GetString(m, v1beta1_api.KeyLastAppliedBackupInvoker)
	invokerKind := GetString(m, v1beta1_api.KeyLastAppliedBackupInvokerKind)

	if data == "" || invokerKind != v1beta1_api.ResourceKindBackupConfiguration {
		return nil, nil
	}
	obj, err := meta.UnmarshalFromJSON([]byte(data), v1beta1_api.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}
	backupConfiguration, ok := obj.(*v1beta1_api.BackupConfiguration)
	if !ok {
		return nil, fmt.Errorf("%s annotations has invalid invoker object", v1beta1_api.KeyLastAppliedBackupInvoker)
	}
	return backupConfiguration, nil
}

// FindBackupConfiguration check if there is any BackupConfiguration exist for a particular workload.
// If multiple BackupConfigurations are found for the workload, it returns error.
func FindBackupConfiguration(lister v1beta1_listers.BackupConfigurationLister, w *wapi.Workload) (*v1beta1_api.BackupConfiguration, error) {
	// list all BackupConfigurations from the lister
	backupConfigurations, err := lister.BackupConfigurations(w.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	result := make([]*v1beta1_api.BackupConfiguration, 0)
	// keep only those BackupConfiguration that has this workload as target
	for _, bc := range backupConfigurations {
		if bc.DeletionTimestamp == nil && IsBackupTarget(bc.Spec.Target, w, bc.Namespace) {
			result = append(result, bc)
		}
	}

	// if there is more than one BackupConfiguration then return error
	if len(result) > 1 {
		var msg bytes.Buffer
		msg.WriteString(fmt.Sprintf("Workload %s/%s matches multiple BackupConfigurations:", w.Namespace, w.Name))
		for i, bc := range result {
			if i > 0 {
				msg.WriteString(", ")
			}
			msg.WriteString(bc.Name)
		}
		return nil, errors.New(msg.String())
	} else if len(result) == 1 {
		// only one BackupConfiguration is found for this workload. So, return it.
		return result[0], nil
	}
	return nil, nil
}

// BackupConfigurationEqual check whether two BackupConfigurations has same specification.
func BackupConfigurationEqual(old, new *v1beta1_api.BackupConfiguration) bool {
	if (old == nil && new != nil) || (old != nil && new == nil) {
		return false
	}
	if old == nil && new == nil {
		return true
	}

	// If "spec.paused" field is changed, we don't need to restart the workload.
	// Hence, we will compare the new and old BackupConfiguration spec after making
	// `spec.paused` field equal. This will avoid the restart.
	// We should not change the original value of "spec.paused" field of the old BackupConfiguration.
	// Hence, we will keep the original value in a temporary variable and re-assign the original value
	// after the comparison.

	oldSpec := &old.Spec
	newSpec := &new.Spec

	oldVal := oldSpec.Paused
	oldSpec.Paused = newSpec.Paused
	result := reflect.DeepEqual(oldSpec, newSpec)
	oldSpec.Paused = oldVal
	return result
}

func BackupPending(phase v1beta1_api.BackupSessionPhase) bool {
	if phase == "" || phase == v1beta1_api.BackupSessionPending {
		return true
	}
	return false
}
