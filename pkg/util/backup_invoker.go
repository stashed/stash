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
	"fmt"
	"reflect"
	"sort"

	v1beta1_api "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	v1beta1_listers "stash.appscode.dev/apimachinery/client/listers/stash/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"kmodules.xyz/client-go/meta"
)

func ExtractAppliedBackupInvokerFromAnnotation(m map[string]string) (unstructured.Unstructured, error) {
	data := GetString(m, v1beta1_api.KeyLastAppliedBackupInvoker)
	if data == "" {
		return unstructured.Unstructured{}, nil
	}

	obj, err := meta.UnmarshalFromJSON([]byte(data), v1beta1_api.SchemeGroupVersion)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&obj)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	return unstructured.Unstructured{Object: u}, nil
}

func FindLatestBackupInvoker(bcLister v1beta1_listers.BackupConfigurationLister, tref v1beta1_api.TargetRef) (unstructured.Unstructured, error) {
	invokers, err := FindBackupInvokers(bcLister, tref)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	// sort the invokers and return the latest one
	if len(invokers) > 0 {
		sort.Slice(invokers, func(i, j int) bool {
			iT := invokers[i].GetCreationTimestamp()
			jT := invokers[j].GetCreationTimestamp()
			if iT.Equal(&jT) {
				return invokers[i].GetName() < invokers[j].GetName()
			}
			return iT.After(jT.Time)
		})
		return invokers[0], nil
	}
	return unstructured.Unstructured{}, nil
}

func FindBackupInvokers(bcLister v1beta1_listers.BackupConfigurationLister, tref v1beta1_api.TargetRef) ([]unstructured.Unstructured, error) {
	invokers := make([]unstructured.Unstructured, 0)
	backupConfigs, err := FindBackupConfiguration(bcLister, tref)
	if err != nil {
		return nil, err
	}
	invokers = append(invokers, backupConfigs...)

	return invokers, nil
}

func FindBackupConfiguration(lister v1beta1_listers.BackupConfigurationLister, tref v1beta1_api.TargetRef) ([]unstructured.Unstructured, error) {
	// list all BackupConfigurations from the lister
	backupConfigurations, err := lister.BackupConfigurations(metav1.NamespaceAll).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	result := make([]unstructured.Unstructured, 0)
	// keep only those BackupConfiguration that has this workload as target
	for _, bc := range backupConfigurations {
		if bc.DeletionTimestamp == nil &&
			IsBackupTarget(bc.Spec.Target, tref, bc.Namespace) &&
			bc.Spec.Driver == v1beta1_api.ResticSnapshotter {
			bc.GetObjectKind().SetGroupVersionKind(v1beta1_api.SchemeGroupVersion.WithKind(v1beta1_api.ResourceKindBackupConfiguration))
			u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(bc)
			if err != nil {
				return nil, err
			}
			result = append(result, unstructured.Unstructured{Object: u})
		}
	}
	return result, nil
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

func InvokerEqual(old, new unstructured.Unstructured) (bool, error) {
	if old.Object == nil && new.Object == nil {
		return true, nil
	}
	if (old.Object == nil && new.Object != nil) ||
		(old.Object != nil && new.Object == nil) {
		return false, nil
	}
	if old.GetKind() != new.GetKind() {
		return false, nil
	}

	switch old.GetKind() {
	case v1beta1_api.ResourceKindBackupConfiguration:
		var oldBC, newBC v1beta1_api.BackupConfiguration
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(old.Object, &oldBC)
		if err != nil {
			return false, err
		}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(new.Object, &newBC)
		if err != nil {
			return false, err
		}
		return BackupConfigurationEqual(&oldBC, &newBC), nil

	case v1beta1_api.ResourceKindRestoreSession:
		var oldRS, newRS v1beta1_api.RestoreSession
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(old.Object, &oldRS)
		if err != nil {
			return false, err
		}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(new.Object, &newRS)
		if err != nil {
			return false, err
		}
		return RestoreSessionEqual(&oldRS, &newRS), nil
	}
	return false, fmt.Errorf("unknown invoker kind: %v", old.GetKind())
}
