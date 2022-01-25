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

package clusterid

import (
	"context"
	"flag"
	"fmt"

	kmapi "kmodules.xyz/client-go/api/v1"

	"github.com/spf13/pflag"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var clusterName = ""

func AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&clusterName, "cluster-name", clusterName, "Name of cluster used in a multi-cluster setup")
}

func AddGoFlags(fs *flag.FlagSet) {
	fs.StringVar(&clusterName, "cluster-name", clusterName, "Name of cluster used in a multi-cluster setup")
}

func ClusterName() string {
	return clusterName
}

func ClusterUID(client corev1.NamespaceInterface) (string, error) {
	ns, err := client.Get(context.TODO(), metav1.NamespaceSystem, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return string(ns.UID), nil
}

func ClusterMetadataForNamespace(ns *core.Namespace) (*kmapi.ClusterMetadata, error) {
	if ns.Name != metav1.NamespaceSystem {
		return nil, fmt.Errorf("expected namespace %s, found namespace %s", metav1.NamespaceSystem, ns.Name)
	}
	name := ns.Annotations[kmapi.ClusterNameKey]
	if name == "" {
		name = ClusterName()
	}
	obj := &kmapi.ClusterMetadata{
		UID:         string(ns.UID),
		Name:        name,
		DisplayName: ns.Annotations[kmapi.ClusterDisplayNameKey],
		Provider:    kmapi.HostingProvider(ns.Annotations[kmapi.ClusterProviderNameKey]),
	}
	return obj, nil
}

func ClusterMetadata(client corev1.NamespaceInterface) (*kmapi.ClusterMetadata, error) {
	ns, err := client.Get(context.TODO(), metav1.NamespaceSystem, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return ClusterMetadataForNamespace(ns)
}
