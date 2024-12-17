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

package cluster

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsOpenClusterHub(mapper meta.RESTMapper) bool {
	if _, err := mapper.RESTMappings(schema.GroupKind{
		Group: "cluster.open-cluster-management.io",
		Kind:  "ManagedCluster",
	}); err == nil {
		return true
	}
	return false
}

func IsOpenClusterSpoke(kc client.Reader) bool {
	var list unstructured.UnstructuredList
	list.SetAPIVersion("operator.open-cluster-management.io/v1")
	list.SetKind("Klusterlet")
	err := kc.List(context.TODO(), &list)
	return err == nil && len(list.Items) > 0
}

func IsOpenClusterMulticlusterControlplane(mapper meta.RESTMapper) bool {
	var missingDeployment bool
	if _, err := mapper.RESTMappings(schema.GroupKind{
		Group: "apps",
		Kind:  "Deployment",
	}); meta.IsNoMatchError(err) {
		missingDeployment = true
	}
	return IsOpenClusterHub(mapper) && missingDeployment
}
