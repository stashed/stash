/*
Copyright The Stash Authors.

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
	"bytes"
	"fmt"
	"reflect"

	"stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	v1beta1_listers "stash.appscode.dev/apimachinery/client/listers/stash/v1beta1"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
	"kmodules.xyz/client-go/meta"
	wapi "kmodules.xyz/webhook-runtime/apis/workload/v1"
)

// GetAppliedBackupBatch check whether BackupBatch was applied as annotation and returns the object definition if exist.
func GetAppliedBackupBatch(m map[string]string) (*v1beta1.BackupBatch, error) {
	data := GetString(m, v1beta1.KeyLastAppliedBackupInvoker)
	invokerKind := GetString(m, v1beta1.KeyLastAppliedBackupInvokerKind)

	if data == "" || invokerKind != v1beta1.ResourceKindBackupBatch {
		return nil, nil
	}
	obj, err := meta.UnmarshalFromJSON([]byte(data), v1beta1.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}
	backupBatch, ok := obj.(*v1beta1.BackupBatch)
	if !ok {
		return nil, fmt.Errorf("%s annotations has invalid invoker object", v1beta1.KeyLastAppliedBackupInvoker)
	}
	return backupBatch, nil
}

// FindBackupBatch check if there is any BackupBatch exist for a particular workload.
// If multiple BackupBatches are found for the workload, it returns error.
func FindBackupBatch(lister v1beta1_listers.BackupBatchLister, w *wapi.Workload) (*v1beta1.BackupBatch, error) {
	// list all BackupBatches from the lister
	backupBatches, err := lister.BackupBatches(w.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	result := make([]*v1beta1.BackupBatch, 0)
	// keep only those BackupBatches that has this workload as target
	for _, backupBatch := range backupBatches {
		for _, member := range backupBatch.Spec.Members {
			if backupBatch.DeletionTimestamp == nil && IsBackupTarget(member.Target, w) {
				result = append(result, backupBatch)
			}
		}
	}

	// if there is more than one BackupBatch then return error
	if len(result) > 1 {
		var msg bytes.Buffer
		msg.WriteString(fmt.Sprintf("Workload %s/%s matches multiple BackupBatches:", w.Namespace, w.Name))
		for i, bc := range result {
			if i > 0 {
				msg.WriteString(", ")
			}
			msg.WriteString(bc.Name)
		}
		return nil, errors.New(msg.String())
	} else if len(result) == 1 {
		// only one BackupBatch is found for this workload. So, return it.
		return result[0], nil
	}
	return nil, nil
}

// BackupBatchEqual check whether two BackupBatches has same specification.
func BackupBatchEqual(old, new *v1beta1.BackupBatch) bool {
	if (old == nil && new != nil) || (old != nil && new == nil) {
		return false
	}
	if old == nil && new == nil {
		return true
	}

	// If "spec.paused" field is changed, we don't need to restart the workload.
	// Hence, we will compare the new and old BackupBatch spec after making
	// `spec.paused` field equal. This will avoid the restart.
	// We should not change the original value of "spec.paused" field of the old BackupBatch.
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
