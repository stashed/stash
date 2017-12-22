package analytics

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"net"
	"sort"

	"github.com/appscode/go/analytics"
	"github.com/appscode/kutil/meta"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func ClientID() string {
	if !meta.PossiblyInCluster() {
		return analytics.ClientID()
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return ""
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return ""
	}
	nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"node-role.kubernetes.io/master": "",
		}).String(),
	})
	if err != nil {
		return reasonForError(err)
	}
	if len(nodes.Items) == 0 {
		nodes, err = client.CoreV1().Nodes().List(metav1.ListOptions{
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
	if len(ips) == 0 {
		return ""
	}
	sort.Slice(ips, func(i, j int) bool { return bytes.Compare(ips[i], ips[j]) < 0 })
	hasher := md5.New()
	for _, ip := range ips {
		hasher.Write(ip)
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
		return string(t.Status().Reason)
	}
	return string(metav1.StatusReasonUnknown)
}
