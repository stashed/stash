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

package siteinfo

import (
	"context"
	"net"
	"strings"

	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/clusterid"
	auditorapi "kmodules.xyz/custom-resources/apis/auditor/v1alpha1"
	"kmodules.xyz/resource-metrics/api"

	"go.bytebuilders.dev/license-verifier/info"
	v "gomodules.xyz/x/version"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GetSiteInfo(cfg *rest.Config, kc kubernetes.Interface, nodes []*core.Node, licenseID string) (*auditorapi.SiteInfo, error) {
	si := auditorapi.SiteInfo{
		TypeMeta: metav1.TypeMeta{
			APIVersion: auditorapi.SchemeGroupVersion.String(),
			Kind:       "SiteInfo",
		},
	}

	if info.ProductName != "" || v.Version.Version != "" || licenseID != "" {
		si.Product = &auditorapi.ProductInfo{}
		si.Product.LicenseID = licenseID
		si.Product.ProductOwnerName = info.ProductOwnerName
		si.Product.ProductOwnerUID = info.ProductOwnerUID
		si.Product.ProductName = info.ProductName
		si.Product.ProductUID = info.ProductUID
		si.Product.Version = auditorapi.Version{
			Version:         v.Version.Version,
			VersionStrategy: v.Version.VersionStrategy,
			CommitHash:      v.Version.CommitHash,
			GitBranch:       v.Version.GitBranch,
			GitTag:          v.Version.GitTag,
			CommitTimestamp: v.Version.CommitTimestamp,
			GoVersion:       v.Version.GoVersion,
			Compiler:        v.Version.Compiler,
			Platform:        v.Version.Platform,
		}
	}

	var err error
	si.Kubernetes.ClusterName = clusterid.ClusterName()
	si.Kubernetes.ClusterUID, err = clusterid.ClusterUID(kc.CoreV1().Namespaces())
	if err != nil {
		return nil, err
	}
	si.Kubernetes.Version, err = kc.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}

	cert, err := meta_util.APIServerCertificate(cfg)
	if err != nil {
		return nil, err
	} else {
		si.Kubernetes.ControlPlane = &auditorapi.ControlPlaneInfo{
			NotBefore: metav1.NewTime(cert.NotBefore),
			NotAfter:  metav1.NewTime(cert.NotAfter),
			// DNSNames:       cert.DNSNames,
			EmailAddresses: cert.EmailAddresses,
			// IPAddresses:    cert.IPAddresses,
			// URIs:           cert.URIs,
		}

		dnsNames := sets.NewString(cert.DNSNames...)
		ips := sets.NewString()
		if len(cert.Subject.CommonName) > 0 {
			if ip := net.ParseIP(cert.Subject.CommonName); ip != nil {
				if !skipIP(ip) {
					ips.Insert(ip.String())
				}
			} else {
				dnsNames.Insert(cert.Subject.CommonName)
			}
		}

		for _, host := range dnsNames.UnsortedList() {
			if host == "kubernetes" ||
				host == "kubernetes.default" ||
				host == "kubernetes.default.svc" ||
				strings.HasSuffix(host, ".svc.cluster.local") ||
				host == "localhost" ||
				!strings.ContainsRune(host, '.') {
				dnsNames.Delete(host)
			}
		}
		si.Kubernetes.ControlPlane.DNSNames = dnsNames.List()

		for _, ip := range cert.IPAddresses {
			if !skipIP(ip) {
				ips.Insert(ip.String())
			}
		}
		si.Kubernetes.ControlPlane.IPAddresses = ips.List()

		uris := make([]string, 0, len(cert.URIs))
		for _, u := range cert.URIs {
			uris = append(uris, u.String())
		}
		si.Kubernetes.ControlPlane.URIs = uris
	}

	if len(nodes) == 0 {
		result, err := kc.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		nodes = make([]*core.Node, len(result.Items))
		for i := range result.Items {
			nodes[i] = &result.Items[i]
		}
	}
	RefreshNodeStats(&si, nodes)

	return &si, nil
}

func RefreshNodeStats(si *auditorapi.SiteInfo, nodes []*core.Node) {
	if len(nodes) == 0 {
		return
	}
	si.Kubernetes.NodeStats.Count = len(nodes)

	var capacity core.ResourceList
	var allocatable core.ResourceList
	for _, node := range nodes {
		capacity = api.AddResourceList(capacity, node.Status.Capacity)
		allocatable = api.AddResourceList(allocatable, node.Status.Allocatable)
	}
	si.Kubernetes.NodeStats.Capacity = capacity
	si.Kubernetes.NodeStats.Allocatable = allocatable
}

func skipIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsMulticast() ||
		ip.IsGlobalUnicast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsLinkLocalUnicast()
}
