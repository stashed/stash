/*
Copyright The Kmodules Authors.

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

package analytics

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"net"
	"os"
	"sort"
	"strings"

	"kmodules.xyz/client-go/meta"

	"github.com/appscode/go/analytics"
	net2 "github.com/appscode/go/net"
	"github.com/appscode/go/sets"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	Key = "APPSCODE_ANALYTICS_CLIENT_ID"
)

func ClientID() string {
	if id, found := os.LookupEnv(Key); found {
		return id
	}

	defer runtime.HandleCrash()

	if !meta.PossiblyInCluster() {
		return analytics.ClientID()
	}

	if ip, err := meta.TestGKE(); err == nil {
		return hash(ip)
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return "$k8s$inclusterconfig"
	}

	if cert, err := meta.APIServerCertificate(cfg); err == nil {
		if domain, err := meta.TestAKS(cert); err == nil {
			return hash(domain)
		}
		if domain, err := meta.TestEKS(cert); err == nil {
			return hash(domain)
		}

		dnsNames := sets.NewString(cert.DNSNames...)

		if len(cert.Subject.CommonName) > 0 {
			if ip := net.ParseIP(cert.Subject.CommonName); ip != nil {
				// GKE
				if !ip.IsLoopback() &&
					!ip.IsMulticast() &&
					!ip.IsGlobalUnicast() &&
					!ip.IsInterfaceLocalMulticast() &&
					!ip.IsLinkLocalMulticast() &&
					!ip.IsLinkLocalUnicast() &&
					(ip.To4() == nil || !net2.IsPrivateIP(ip)) { // either ipv6 or a public ipv4
					return hash(cert.Subject.CommonName)
				}
			} else {
				dnsNames.Insert(cert.Subject.CommonName)
			}
		}

		var domains []string
		for _, host := range dnsNames.List() {
			if host == "kubernetes" ||
				host == "kubernetes.default" ||
				host == "kubernetes.default.svc" ||
				host == "kubernetes.default.svc.cluster.local" ||
				host == "localhost" ||
				strings.HasSuffix(host, ".compute.internal") ||
				!strings.ContainsRune(host, '.') {
				continue
			}
			domains = append(domains, host)
		}
		if len(domains) > 0 {
			sort.Strings(domains)
			return hash(domains...)
		}

		var ips []net.IP
		for i, ip := range cert.IPAddresses {
			if ip.IsLoopback() ||
				ip.IsMulticast() ||
				ip.IsGlobalUnicast() ||
				ip.IsInterfaceLocalMulticast() ||
				ip.IsLinkLocalMulticast() ||
				ip.IsLinkLocalUnicast() {
				continue
			}
			if ip.To4() == nil || !net2.IsPrivateIP(ip) {
				ips = append(ips, cert.IPAddresses[i])
			}
		}
		if len(ips) > 0 {
			sort.Slice(ips, func(i, j int) bool { return bytes.Compare(ips[i], ips[j]) < 0 })
			hasher := md5.New()
			for _, ip := range ips {
				_, _ = hasher.Write(ip)
			}
			return hex.EncodeToString(hasher.Sum(nil))
		}
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "$k8s$newforconfig"
	}
	nodes, err := client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/master",
	})
	if err != nil {
		return reasonForError(err)
	}
	if len(nodes.Items) == 0 {
		nodes, err = client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"kubernetes.io/hostname": "minikube",
			}).String(),
		})
		if err != nil {
			return reasonForError(err)
		}
	}

	ips := make([]net.IP, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		ip := nodeIP(node)
		if ip != nil {
			ips = append(ips, ip)
		}
	}
	sort.Slice(ips, func(i, j int) bool { return bytes.Compare(ips[i], ips[j]) < 0 })
	hasher := md5.New()
	for _, ip := range ips {
		_, _ = hasher.Write(ip)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func hash(data ...string) string {
	hasher := md5.New()
	for _, x := range data {
		_, _ = hasher.Write([]byte(x))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func nodeIP(node core.Node) []byte {
	for _, addr := range node.Status.Addresses {
		if addr.Type == core.NodeExternalIP {
			return ipBytes(net.ParseIP(addr.Address))
		}
	}
	for _, addr := range node.Status.Addresses {
		if addr.Type == core.NodeInternalIP {
			return ipBytes(net.ParseIP(addr.Address))
		}
	}
	return nil
}

func ipBytes(ip net.IP) []byte {
	if ip == nil {
		return nil
	}
	v4 := ip.To4()
	if v4 != nil {
		return v4
	}
	v6 := ip.To16()
	if v6 != nil {
		return v6
	}
	return nil
}

func reasonForError(err error) string {
	switch t := err.(type) {
	case kerr.APIStatus:
		return "$k8s$err$" + string(t.Status().Reason)
	}
	return "$k8s$err$" + trim(err.Error(), 32) // 32 = length of uuid
}

func trim(s string, length int) string {
	if len(s) > length {
		return s[:length]
	}
	return s
}
