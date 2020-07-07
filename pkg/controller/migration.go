/*
Copyright AppsCode Inc. and Contributors

Licensed under the PolyForm Noncommercial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/PolyForm-Noncommercial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"

	"github.com/appscode/go/encoding/json/types"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/dynamic"
)

func (c *StashController) MigrateObservedGeneration() error {
	dc, err := dynamic.NewForConfig(c.clientConfig)
	if err != nil {
		return err
	}

	var errs []error
	for _, gvr := range []schema.GroupVersionResource{
		v1alpha1.SchemeGroupVersion.WithResource(v1alpha1.ResourcePluralRepository),
		v1alpha1.SchemeGroupVersion.WithResource(v1alpha1.ResourcePluralRecovery),
	} {
		client := dc.Resource(gvr)
		objects, err := client.Namespace(core.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			errs = append(errs, err)
			continue
			// return err
		}
		for _, obj := range objects.Items {
			changed1, e1 := convertObservedGenerationToInt64(&obj)
			changed2, e2 := moveStatusSize(&obj)
			if e1 != nil || e2 != nil {
				errs = append(errs, e1, e2)
			} else if changed1 || changed2 {
				_, e3 := client.Namespace(obj.GetNamespace()).UpdateStatus(context.TODO(), &obj, metav1.UpdateOptions{})
				errs = append(errs, e3)
			}
		}
	}
	return utilerrors.NewAggregate(errs)
}

func convertObservedGenerationToInt64(u *unstructured.Unstructured) (bool, error) {
	val, found, err := unstructured.NestedFieldNoCopy(u.Object, "status", "observedGeneration")
	if err != nil {
		return false, err
	}
	if found {
		if _, ok := val.(string); ok {
			observed, err := types.ParseIntHash(val)
			if err != nil {
				return false, err
			}
			err = unstructured.SetNestedField(u.Object, observed.Generation(), "status", "observedGeneration")
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func moveStatusSize(u *unstructured.Unstructured) (bool, error) {
	val, found, err := unstructured.NestedFieldNoCopy(u.Object, "status", "size")
	if err != nil {
		return false, err
	}
	if found {
		err = unstructured.SetNestedField(u.Object, val, "status", "totalSize")
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}
