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

func ExtractAppliedRestoreInvokerFromAnnotation(m map[string]string) (unstructured.Unstructured, error) {
	data := GetString(m, v1beta1_api.KeyLastAppliedRestoreInvoker)
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

func FindLatestRestoreInvoker(rsLister v1beta1_listers.RestoreSessionLister, tref v1beta1_api.TargetRef) (unstructured.Unstructured, error) {
	invokers, err := FindRestoreInvokers(rsLister, tref)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	precedence := map[v1beta1_api.RestorePhase]int{
		v1beta1_api.RestoreRunning: 0,
		v1beta1_api.RestorePending: 1,
		v1beta1_api.RestoreFailed:  2,
	}

	// sort the invokers and return the latest one
	if len(invokers) > 0 {
		sort.Slice(invokers, func(i, j int) bool {
			iPhase, _, _ := unstructured.NestedFieldCopy(invokers[i].Object, "status", "phase")
			jPhase, _, _ := unstructured.NestedFieldCopy(invokers[j].Object, "status", "phase")

			if iPhase != jPhase && iPhase != "" && jPhase != "" {
				return precedence[iPhase.(v1beta1_api.RestorePhase)] < precedence[jPhase.(v1beta1_api.RestorePhase)]
			}

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

func FindRestoreInvokers(rsLister v1beta1_listers.RestoreSessionLister, tref v1beta1_api.TargetRef) ([]unstructured.Unstructured, error) {
	invokers := make([]unstructured.Unstructured, 0)
	restoreSessions, err := FindRestoreSession(rsLister, tref)
	if err != nil {
		return nil, err
	}
	invokers = append(invokers, restoreSessions...)
	return invokers, nil
}

func FindRestoreSession(lister v1beta1_listers.RestoreSessionLister, targetRef v1beta1_api.TargetRef) ([]unstructured.Unstructured, error) {
	// list all RestoreSessions from the lister
	restoreSessions, err := lister.RestoreSessions(metav1.NamespaceAll).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	result := make([]unstructured.Unstructured, 0)
	// keep only those RestoreSession that has this workload as target
	for _, rs := range restoreSessions {
		if rs.DeletionTimestamp == nil &&
			IsRestoreTarget(rs.Spec.Target, targetRef, rs.Namespace) &&
			rs.Spec.Driver == v1beta1_api.ResticSnapshotter {
			rs.GetObjectKind().SetGroupVersionKind(v1beta1_api.SchemeGroupVersion.WithKind(v1beta1_api.ResourceKindRestoreSession))
			u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(rs)
			if err != nil {
				return nil, err
			}
			result = append(result, unstructured.Unstructured{Object: u})
		}
	}
	return result, nil
}

// RestoreSessionEqual check whether two RestoreSessions has same specification.
func RestoreSessionEqual(old, new *v1beta1_api.RestoreSession) bool {
	var oldSpec, newSpec *v1beta1_api.RestoreSessionSpec
	var oldName, newName string

	if old != nil {
		oldSpec = &old.Spec
		oldName = old.Name
	}
	if new != nil {
		newSpec = &new.Spec
		newName = new.Name
	}

	// user may create new RestoreSession with same spec. in this case, spec will be same but name will be different
	if oldName != newName {
		return false
	}

	// user may update existing RestoreSession spec. so, we need to compare new and old specification
	return reflect.DeepEqual(oldSpec, newSpec)
}
