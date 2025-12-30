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
	"fmt"
	"strings"

	kmapi "kmodules.xyz/client-go/api/v1"

	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/klog/v2"
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
	if err != nil {
		if !meta.IsNoMatchError(err) && !apierrors.IsNotFound(err) {
			panic(err) // panic if 403 (missing rbac)
		}
	}
	return len(list.Items) > 0
}

func IsACEManagedSpoke(kc client.Reader) bool {
	if !IsOpenClusterSpoke(kc) {
		return false
	}

	var list unstructured.UnstructuredList
	list.SetAPIVersion("cluster.open-cluster-management.io/v1alpha1")
	list.SetKind("ClusterClaim")
	err := kc.List(context.TODO(), &list)
	if err != nil {
		klog.Errorln(err)
	}

	n := 0
	for _, item := range list.Items {
		if item.GetName() == kmapi.ClusterClaimKeyID ||
			item.GetName() == kmapi.ClusterClaimKeyInfo {
			n++
		}
		if n == 2 {
			return true
		}
	}
	return false
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

type ClientOrgResult struct {
	IsClientOrg bool
	OrgID       string
	Namespace   core.Namespace
}

func IsClientOrgMember(kc client.Client, user user.Info) (*ClientOrgResult, error) {
	orgs, exists := user.GetExtra()[kmapi.AceOrgIDKey]
	if !exists || len(orgs) == 0 {
		return &ClientOrgResult{}, nil
	}
	if len(orgs) > 1 {
		return nil, fmt.Errorf("user %s associated with multiple orgs %v", user.GetName(), orgs)
	}

	var list core.NamespaceList
	if err := kc.List(context.TODO(), &list, client.MatchingLabels{
		kmapi.ClientOrgKey: "true",
	}); err != nil {
		return nil, err
	}

	for _, ns := range list.Items {
		if ns.Annotations[kmapi.AceOrgIDKey] == orgs[0] {
			return &ClientOrgResult{
				IsClientOrg: true,
				OrgID:       orgs[0],
				Namespace:   ns,
			}, nil
		}
	}
	return &ClientOrgResult{}, nil
}

func ClientDashboardTitle(title string) string {
	title = strings.TrimPrefix(title, "KubeDB /")
	return strings.TrimSpace(title)
}
