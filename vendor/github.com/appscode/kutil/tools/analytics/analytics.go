package analytics

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
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
	if id, found := os.LookupEnv(Key); found {
		return id
	}

	defer runtime.HandleCrash()

	if !meta.PossiblyInCluster() {
		return analytics.ClientID()
	}

	if uid, err := tryGKE(); err == nil {
		return uid
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

// Product file path that contains the cloud service name.
// This is a variable instead of a const to enable testing.
var gceProductNameFile = "/sys/class/dmi/id/product_name"

// ref: https://cloud.google.com/compute/docs/storing-retrieving-metadata
func tryGKE() (string, error) {
	// ref: https://github.com/kubernetes/kubernetes/blob/a0f94123616c275f94e7a5b680d60d6f34e92f37/pkg/credentialprovider/gcp/metadata.go#L115
	data, err := ioutil.ReadFile(gceProductNameFile)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(string(data))
	if name != "Google" && name != "Google Compute Engine" {
		return "", errors.New("not GKE")
	}

	client := &http.Client{Timeout: time.Millisecond * 100}
	req, err := http.NewRequest(http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/attributes/kube-env", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	content := make(map[string]interface{})
	err = yaml.Unmarshal(body, &content)
	if err != nil {
		return "", err
	}
	v, ok := content["KUBERNETES_MASTER_NAME"]
	if !ok {
		return "", errors.New("missing  KUBERNETES_MASTER_NAME")
	}
	masterIP := v.(string)
	hashed := md5.Sum([]byte(masterIP))
	return hex.EncodeToString(hashed[:]), nil
}

func trim(s string, length int) string {
	if len(s) > length {
		return s[:length]
	}
	return s
}
