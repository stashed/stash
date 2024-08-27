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
	"strings"

	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsVirtualCluster(kc client.Client) (bool, error) {
	var list core.NodeList
	err := kc.List(context.TODO(), &list)
	if err != nil {
		return false, err
	}
	for _, node := range list.Items {
		_, f1 := node.Annotations["vcluster.loft.sh/managed-annotations"]
		_, f2 := node.Annotations["vcluster.loft.sh/managed-labels"]
		for _, addr := range node.Status.Addresses {
			if addr.Type == core.NodeHostName {
				if f1 && f2 && strings.HasSuffix(addr.Address, ".nodes.vcluster.com") {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func MustIsVirtualCluster(kc client.Client) bool {
	ok, _ := IsVirtualCluster(kc)
	return ok
}
