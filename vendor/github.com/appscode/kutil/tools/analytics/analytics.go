package analytics

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/appscode/go/analytics"
	"github.com/appscode/kutil/meta"
	"github.com/ghodss/yaml"
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
	defer runtime.HandleCrash()

	if !meta.PossiblyInCluster() {
		return analytics.ClientID()
	}

	if id := os.Getenv(Key); id != "" {
		return id
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return "$k8s$inclusterconfig"
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "$k8s$newforconfig"
	}
	nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/master",
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
		return tryGKE()
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
		return "$k8s$err$" + string(t.Status().Reason)
	}
	return "$k8s$err$" + trim(err.Error(), 32) // 32 = length of uuid
}

// ref: https://cloud.google.com/compute/docs/storing-retrieving-metadata
func tryGKE() string {
	client := &http.Client{Timeout: time.Millisecond * 100}
	req, err := http.NewRequest(http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/attributes/kube-env", nil)
	if err != nil {
		return "$gke$err$" + trim(err.Error(), 32) // 32 = length of uuid
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := client.Do(req)
	if err != nil {
		return "$gke$err$" + trim(err.Error(), 32) // 32 = length of uuid
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "$gke$ReadAll(resp.Body)"
	}
	data := make(map[string]interface{})
	err = yaml.Unmarshal(body, &data)
	if err != nil {
		return "$gke$Unmarshal(body)"
	}
	v, ok := data["KUBERNETES_MASTER_NAME"]
	if !ok {
		return "$gke$KUBERNETES_MASTER_NAME"
	}
	masterIP := v.(string)
	hashed := md5.Sum([]byte(masterIP))
	return hex.EncodeToString(hashed[:])
}

func trim(s string, length int) string {
	if len(s) > length {
		return s[:length]
	}
	return s
}
