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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	kmapi "kmodules.xyz/client-go/api/v1"
	cu "kmodules.xyz/client-go/client"

	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ClusterUID(c client.Reader) (string, error) {
	var ns core.Namespace
	err := c.Get(context.TODO(), client.ObjectKey{Name: metav1.NamespaceSystem}, &ns)
	if err != nil {
		return "", err
	}
	return string(ns.UID), nil
}

func ClusterMetadata(c client.Reader) (*kmapi.ClusterMetadata, error) {
	var ns core.Namespace
	err := c.Get(context.TODO(), client.ObjectKey{Name: metav1.NamespaceSystem}, &ns)
	if err != nil {
		return nil, err
	}

	var cm core.ConfigMap
	err = c.Get(context.TODO(), client.ObjectKey{Name: kmapi.AceInfoConfigMapName, Namespace: metav1.NamespacePublic}, &cm)
	if err == nil {
		result, err := ClusterMetadataFromConfigMap(&cm, string(ns.UID))
		if err == nil {
			return result, nil
		}
	} else if !kerr.IsNotFound(err) {
		return nil, err
	}

	return LegacyClusterMetadataFromNamespace(&ns)
}

func LegacyClusterMetadataFromNamespace(ns *core.Namespace) (*kmapi.ClusterMetadata, error) {
	if ns.Name != metav1.NamespaceSystem {
		return nil, fmt.Errorf("expected namespace %s, found namespace %s", metav1.NamespaceSystem, ns.Name)
	}
	name := ns.Annotations[kmapi.ClusterNameKey]
	if name == "" {
		name = ClusterName()
	}
	md := &kmapi.ClusterMetadata{
		UID:         string(ns.UID),
		Name:        name,
		DisplayName: ns.Annotations[kmapi.ClusterDisplayNameKey],
		Provider:    kmapi.HostingProvider(ns.Annotations[kmapi.ClusterProviderNameKey]),
	}
	return md, nil
}

func ClusterMetadataFromConfigMap(cm *core.ConfigMap, clusterUIDVerifier string) (*kmapi.ClusterMetadata, error) {
	if cm.Name != kmapi.AceInfoConfigMapName || cm.Namespace != metav1.NamespacePublic {
		return nil, fmt.Errorf("expected configmap %s/%s, found %s/%s", metav1.NamespacePublic, kmapi.AceInfoConfigMapName, cm.Namespace, cm.Name)
	}

	md := &kmapi.ClusterMetadata{
		UID:         cm.Data["uid"],
		Name:        cm.Data["name"],
		DisplayName: cm.Data["displayName"],
		Provider:    kmapi.HostingProvider(cm.Data["provider"]),
		OwnerID:     cm.Data["ownerID"],
		OwnerType:   cm.Data["ownerType"],
		APIEndpoint: cm.Data["apiEndpoint"],
		CABundle:    cm.Data["ca.crt"],
	}

	data, err := json.Marshal(md)
	if err != nil {
		return nil, err
	}
	hasher := hmac.New(sha256.New, []byte(clusterUIDVerifier))
	hasher.Write(data)
	messageMAC := hasher.Sum(nil)
	expectedMAC := cm.BinaryData["mac"]
	if !hmac.Equal(messageMAC, expectedMAC) {
		return nil, fmt.Errorf("configmap %s/%s fails validation", cm.Namespace, cm.Name)
	}

	if md.Name == "" {
		md.Name = ClusterName()
	}
	return md, nil
}

func UpsertClusterMetadata(kc client.Client, md *kmapi.ClusterMetadata) error {
	obj := core.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kmapi.AceInfoConfigMapName,
			Namespace: metav1.NamespacePublic,
		},
	}

	data, err := json.Marshal(md)
	if err != nil {
		return err
	}
	hasher := hmac.New(sha256.New, []byte(md.UID))
	hasher.Write(data)
	messageMAC := hasher.Sum(nil)

	_, err = cu.CreateOrPatch(context.TODO(), kc, &obj, func(o client.Object, createOp bool) client.Object {
		cm := o.(*core.ConfigMap)
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}

		cm.Data["uid"] = md.UID
		cm.Data["name"] = md.Name
		cm.Data["displayName"] = md.DisplayName
		cm.Data["provider"] = string(md.Provider)
		cm.Data["ownerID"] = md.OwnerID
		cm.Data["ownerType"] = md.OwnerType
		cm.Data["apiEndpoint"] = md.APIEndpoint
		cm.Data["ca.crt"] = md.CABundle

		cm.BinaryData = map[string][]byte{
			"mac": messageMAC,
		}
		return cm
	})
	return err
}

func DetectCAPICluster(kc client.Reader) (*kmapi.CAPIClusterInfo, error) {
	var list unstructured.UnstructuredList
	list.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "cluster.x-k8s.io",
		Version: "v1beta1",
		Kind:    "Cluster",
	})
	err := kc.List(context.TODO(), &list)
	if meta.IsNoMatchError(err) || len(list.Items) == 0 {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else if len(list.Items) > 1 {
		klog.Warningln("multiple CAPI cluster object found")
		return nil, nil
	}

	obj := list.Items[0].UnstructuredContent()
	capiProvider, clusterName, ns, err := getCAPIValues(obj)
	if err != nil {
		return nil, err
	}

	return &kmapi.CAPIClusterInfo{
		Provider:    getProviderName(capiProvider),
		Namespace:   ns,
		ClusterName: clusterName,
	}, nil
}

func getCAPIValues(values map[string]any) (string, string, string, error) {
	capiProvider, ok, err := unstructured.NestedString(values, "spec", "infrastructureRef", "kind")
	if err != nil {
		return "", "", "", err
	} else if !ok || capiProvider == "" {
		return "", "", "", nil
	}

	clusterName, ok, err := unstructured.NestedString(values, "metadata", "name")
	if err != nil {
		return "", "", "", err
	} else if !ok || clusterName == "" {
		return "", "", "", nil
	}

	ns, ok, err := unstructured.NestedString(values, "metadata", "namespace")
	if err != nil {
		return "", "", "", err
	} else if !ok || ns == "" {
		return "", "", "", nil
	}

	return capiProvider, clusterName, ns, nil
}

func getProviderName(kind string) kmapi.CAPIProvider {
	switch kind {
	case "AWSManagedCluster", "AWSManagedControlPlane":
		return kmapi.CAPIProviderCAPA
	case "AzureManagedCluster":
		return kmapi.CAPIProviderCAPZ
	case "GCPManagedCluster":
		return kmapi.CAPIProviderCAPG
	case "HetznerCluster":
		return kmapi.CAPIProviderCAPH
	case "KubevirtCluster":
		return kmapi.CAPIProviderCAPK
	}
	return ""
}

func DetectClusterManager(kc client.Client, mappers ...meta.RESTMapper) kmapi.ClusterManager {
	mapper := kc.RESTMapper()
	if len(mappers) > 0 {
		mapper = mappers[0]
	}

	var result kmapi.ClusterManager
	if IsACEManaged(kc) {
		result |= kmapi.ClusterManagerACE
	}
	if IsOpenClusterHub(mapper) {
		result |= kmapi.ClusterManagerOCMHub
	}
	if IsOpenClusterSpoke(kc) {
		result |= kmapi.ClusterManagerOCMSpoke
	}
	if IsOpenClusterMulticlusterControlplane(mapper) {
		result |= kmapi.ClusterManagerOCMMulticlusterControlplane
	}
	if IsRancherManaged(mapper) {
		result |= kmapi.ClusterManagerRancher
	}
	if IsOpenShiftManaged(mapper) {
		result |= kmapi.ClusterManagerOpenShift
	}
	if MustIsVirtualCluster(kc) {
		result |= kmapi.ClusterManagerVirtualCluster
	}
	return result
}

func IsDefault(kc client.Client, cm kmapi.ClusterManager, gvk schema.GroupVersionKind, key types.NamespacedName) (bool, error) {
	if cm.ManagedByRancher() {
		return IsInSystemProject(kc, key.Namespace)
	}
	return IsSingletonResource(kc, gvk, key)
}

func IsSingletonResource(kc client.Client, gvk schema.GroupVersionKind, key types.NamespacedName) (bool, error) {
	var list unstructured.UnstructuredList
	list.SetGroupVersionKind(gvk)
	err := kc.List(context.TODO(), &list, client.InNamespace(key.Namespace))
	if err != nil {
		return false, err
	}
	return len(list.Items) == 1, nil
}
