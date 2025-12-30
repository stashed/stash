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
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"sigs.k8s.io/yaml"
)

// ref: https://cloud.google.com/compute/docs/storing-retrieving-metadata
func TestGKE() (string, error) {
	// ref: https://github.com/kubernetes/kubernetes/blob/a0f94123616c275f94e7a5b680d60d6f34e92f37/pkg/credentialprovider/gcp/metadata.go#L115
	data, err := os.ReadFile("/sys/class/dmi/id/product_name")
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
	defer resp.Body.Close() // nolint:errcheck
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	content := make(map[string]any)
	err = yaml.Unmarshal(body, &content)
	if err != nil {
		return "", err
	}
	v, ok := content["KUBERNETES_MASTER_NAME"]
	if !ok {
		return "", errors.New("missing  KUBERNETES_MASTER_NAME")
	}
	return v.(string), nil
}

func TestAKS(cert *x509.Certificate) (string, error) {
	for _, host := range cert.DNSNames {
		if strings.HasSuffix(host, aksDomain) && isAKS() == nil {
			return host, nil
		}
	}
	return "", errors.New("not AKS")
}

// ref: https://cloud.google.com/compute/docs/storing-retrieving-metadata
func isAKS() error {
	data, err := os.ReadFile("/sys/class/dmi/id/sys_vendor")
	if err != nil {
		return err
	}
	sysVendor := strings.TrimSpace(string(data))

	data, err = os.ReadFile("/sys/class/dmi/id/product_name")
	if err != nil {
		return err
	}
	productName := strings.TrimSpace(string(data))

	if sysVendor != "Microsoft Corporation" && productName != "Virtual Machine" {
		return errors.New("not AKS")
	}
	return nil
}

func TestEKS(cert *x509.Certificate) (string, error) {
	for _, host := range cert.DNSNames {
		if strings.HasSuffix(host, eksDomain) && isEKS() == nil {
			return host, nil
		}
	}
	return "", errors.New("not EKS")
}

// ref: https://cloud.google.com/compute/docs/storing-retrieving-metadata
func isEKS() error {
	data, err := os.ReadFile("/sys/class/dmi/id/sys_vendor")
	if err != nil {
		return err
	}
	sysVendor := strings.TrimSpace(string(data))

	if sysVendor != "Amazon EC2" {
		return errors.New("not EKS")
	}
	return nil
}
