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

	kmapi "kmodules.xyz/client-go/api/v1"
	clustermeta "kmodules.xyz/client-go/cluster"

	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func ClusterUID(client corev1.NamespaceInterface) (string, error) {
	ns, err := client.Get(context.TODO(), metav1.NamespaceSystem, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return string(ns.UID), nil
}

func ClusterMetadata(client kubernetes.Interface) (*kmapi.ClusterMetadata, error) {
	ns, err := client.CoreV1().Namespaces().Get(context.TODO(), metav1.NamespaceSystem, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	cm, err := client.CoreV1().ConfigMaps(metav1.NamespacePublic).Get(context.TODO(), kmapi.AceInfoConfigMapName, metav1.GetOptions{})
	if err == nil {
		result, err := clustermeta.ClusterMetadataFromConfigMap(cm, clustermeta.DetectClusterMode(ns), string(ns.UID))
		if err == nil {
			return result, nil
		}
	} else if !kerr.IsNotFound(err) {
		return nil, err
	}

	return clustermeta.LegacyClusterMetadataFromNamespace(ns)
}
