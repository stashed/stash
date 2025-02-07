/*
Copyright AppsCode Inc. and Contributors.

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

package identity

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"

	kmapi "kmodules.xyz/client-go/api/v1"
	clustermeta "kmodules.xyz/client-go/cluster"
	identityapi "kmodules.xyz/resource-metadata/apis/identity/v1alpha1"

	"go.bytebuilders.dev/license-verifier/info"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/klog/v2"
	"moul.io/http2curl/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	baseURL string
	token   string
	caCert  []byte
	client  *http.Client

	kc client.Reader
}

func NewClient(baseURL, token string, caCert []byte, kc client.Reader) (*Client, error) {
	c := &Client{
		baseURL: baseURL,
		token:   token,
		caCert:  caCert,
		kc:      kc,
	}
	if len(caCert) == 0 {
		u, err := url.Parse(baseURL)
		if err != nil {
			return nil, err
		}
		// use InsecureSkipVerify, if IP address is used for baseURL host
		if ip := net.ParseIP(u.Hostname()); ip != nil && u.Scheme == "https" {
			customTransport := http.DefaultTransport.(*http.Transport).Clone()
			customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			c.client = &http.Client{Transport: customTransport}
		} else {
			c.client = http.DefaultClient
		}
	} else {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		tlsConfig := &tls.Config{
			RootCAs: caCertPool,
		}
		transport := &http.Transport{TLSClientConfig: tlsConfig}
		c.client = &http.Client{Transport: transport}
	}
	return c, nil
}

func (c *Client) Identify(clusterUID string) (*kmapi.ClusterMetadata, error) {
	u, err := info.APIServerAddress(c.baseURL)
	if err != nil {
		return nil, err
	}
	apiEndpoint := u.String()
	u.Path = path.Join(u.Path, "api/v1/clustersv2/identity", clusterUID)

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// add authorization header to the req
	if c.token != "" {
		req.Header.Add("Authorization", "Bearer "+c.token)
	}
	if klog.V(8).Enabled() {
		command, _ := http2curl.GetCurlCommand(req)
		klog.V(8).Infoln(command.String())
	}

	resp, err := c.client.Do(req)
	if err != nil {
		var ce *tls.CertificateVerificationError
		if errors.As(err, &ce) {
			klog.ErrorS(err, "UnverifiedCertificates")
			for _, cert := range ce.UnverifiedCertificates {
				klog.Errorln(string(encodeCertPEM(cert)))
			}
		}
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, apierrors.NewGenericServerResponse(
			resp.StatusCode,
			http.MethodGet,
			schema.GroupResource{Group: identityapi.GroupName, Resource: identityapi.ResourceClusterIdentities},
			"",
			string(body),
			0,
			false,
		)
	}
	var md kmapi.ClusterMetadata
	err = json.Unmarshal(body, &md)
	if err != nil {
		return nil, err
	}

	md.APIEndpoint = apiEndpoint
	md.CABundle = string(c.caCert)

	return &md, nil
}

func (c *Client) GetToken() (*identityapi.InboxTokenRequestResponse, error) {
	u, err := info.APIServerAddress(c.baseURL)
	if err != nil {
		return nil, err
	}

	id, err := c.GetIdentity()
	if err != nil {
		return nil, err
	}

	u.Path = path.Join(u.Path, "api/v1/agent", id.Status.Name, id.Status.UID, "token")

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// add authorization header to the req
	if c.token != "" {
		req.Header.Add("Authorization", "Bearer "+c.token)
	}
	if klog.V(8).Enabled() {
		command, _ := http2curl.GetCurlCommand(req)
		klog.V(8).Infoln(command.String())
	}

	resp, err := c.client.Do(req)
	if err != nil {
		var ce *tls.CertificateVerificationError
		if errors.As(err, &ce) {
			klog.ErrorS(err, "UnverifiedCertificates")
			for _, cert := range ce.UnverifiedCertificates {
				klog.Errorln(string(encodeCertPEM(cert)))
			}
		}
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	tokenResponse := &identityapi.InboxTokenRequestResponse{}
	if err = json.Unmarshal(body, tokenResponse); err != nil {
		return nil, err
	}
	tokenResponse.CABundle = string(c.caCert)

	return tokenResponse, nil
}

const SelfName = "self"

func (c *Client) GetIdentity() (*identityapi.ClusterIdentity, error) {
	md, err := clustermeta.ClusterMetadata(c.kc)
	if err != nil {
		return nil, err
	}
	return &identityapi.ClusterIdentity{
		ObjectMeta: metav1.ObjectMeta{
			UID:        types.UID("cid-" + md.UID),
			Name:       SelfName,
			Generation: 1,
		},
		Status: *md,
	}, nil
}

func encodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}
