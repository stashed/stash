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
package controller

import (
	"stash.appscode.dev/stash/apis/stash/v1alpha1"

	"github.com/appscode/go/encoding/json/types"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

func (c *StashController) MigrateObservedGeneration() error {
	dc, err := dynamic.NewForConfig(c.clientConfig)
	if err != nil {
		return err
	}

	repoClient := dc.Resource(v1alpha1.SchemeGroupVersion.WithResource(v1alpha1.ResourcePluralRepository))
	repos, err := repoClient.Namespace(core.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, repo := range repos.Items {
		err := convertObservedGenerationToInt64(repoClient, repo)
		if err != nil {
			return err
		}
	}

	return nil
}

func convertObservedGenerationToInt64(client dynamic.NamespaceableResourceInterface, u unstructured.Unstructured) error {
	val, found, err := unstructured.NestedFieldNoCopy(u.Object, "status", "observedGeneration")
	if err != nil {
		return err
	}
	if found {
		if _, ok := val.(string); ok {
			observed, err := types.ParseIntHash(val)
			if err != nil {
				return err
			}
			err = unstructured.SetNestedField(u.Object, observed.Generation(), "status", "observedGeneration")
			if err != nil {
				return err
			}
			_, err = client.Namespace(u.GetNamespace()).UpdateStatus(&u, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
