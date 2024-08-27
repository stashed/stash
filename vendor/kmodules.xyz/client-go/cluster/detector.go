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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	kmapi "kmodules.xyz/client-go/api/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

const (
	aksDomain      = ".azmk8s.io"
	eksDomain      = ".eks.amazonaws.com"
	exoscaleDomain = ".exo.io"
	doDomain       = ".k8s.ondigitalocean.com"
	lkeDomain      = ".linodelke.net"
	scalewayDomain = ".scw.cloud"
	vultrDomain    = ".vultr-k8s.com"
)

func APIServerCertificate(cfg *rest.Config) (*x509.Certificate, error) {
	err := rest.LoadTLSFiles(cfg)
	if err != nil {
		return nil, err
	}

	// create ca cert pool
	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(cfg.CAData)
	if !ok {
		return nil, fmt.Errorf("can't append caCert to caCertPool")
	}

	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{RootCAs: caCertPool},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(cfg.Host)
	if err != nil {
		return nil, err
	}
	for i := range resp.TLS.VerifiedChains {
		return resp.TLS.VerifiedChains[i][0], nil
	}
	return nil, fmt.Errorf("no cert found")
}

func DetectProvider(cfg *rest.Config, mapper meta.RESTMapper) (kmapi.HostingProvider, error) {
	crt, err := APIServerCertificate(cfg)
	if err != nil {
		return "", err
	}

	for _, host := range crt.DNSNames {
		if strings.HasSuffix(host, eksDomain) {
			return kmapi.HostingProviderAWS, nil
		} else if strings.HasSuffix(host, aksDomain) {
			return kmapi.HostingProviderAzure, nil
		} else if strings.HasSuffix(host, doDomain) {
			return kmapi.HostingProviderDigitalOcean, nil
		} else if strings.HasSuffix(host, exoscaleDomain) {
			return kmapi.HostingProviderExoscale, nil
		} else if strings.HasSuffix(host, lkeDomain) {
			return kmapi.HostingProviderLinode, nil
		} else if strings.HasSuffix(host, scalewayDomain) {
			return kmapi.HostingProviderScaleway, nil
		} else if strings.HasSuffix(host, vultrDomain) {
			return kmapi.HostingProviderVultr, nil
		}
	}

	// GKE does not use any custom domain
	if _, err := mapper.RESTMappings(schema.GroupKind{
		Group: "networking.gke.io",
		Kind:  "Network",
	}); err == nil {
		return kmapi.HostingProviderGoogleCloud, nil
	}

	return "", nil
}
