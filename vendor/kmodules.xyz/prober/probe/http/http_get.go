/*
Copyright 2015 The Kubernetes Authors.

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

package http

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	api "kmodules.xyz/prober/api"

	utilnet "k8s.io/apimachinery/pkg/util/net"
)

const (
	maxRespBodyLength = 10 * 1 << 10 // 10KB
)

// New creates GetProber that will skip TLS verification while probing.
// followNonLocalRedirects configures whether the prober should follow redirects to a different hostname.
//   If disabled, redirects to other hosts will trigger a warning result.
func NewHttpGet(followNonLocalRedirects bool) GetProber {
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	return NewGetWithTLSConfig(tlsConfig, followNonLocalRedirects)
}

// NewWithTLSConfig takes tls config as parameter.
// followNonLocalRedirects configures whether the prober should follow redirects to a different hostname.
//   If disabled, redirects to other hosts will trigger a warning result.
func NewGetWithTLSConfig(config *tls.Config, followNonLocalRedirects bool) GetProber {
	// We do not want the probe use node's local proxy set.
	transport := utilnet.SetTransportDefaults(
		&http.Transport{
			TLSClientConfig:   config,
			DisableKeepAlives: true,
			Proxy:             http.ProxyURL(nil),
		})
	return httpGetProber{transport, followNonLocalRedirects}
}

// GetProber is an interface that defines the Probe function for doing HTTP probe.
type GetProber interface {
	Probe(url *url.URL, headers http.Header, timeout time.Duration) (api.Result, string, error)
}

type httpGetProber struct {
	transport               *http.Transport
	followNonLocalRedirects bool
}

// Probe returns a ProbeRunner capable of running an HTTP check.
func (pr httpGetProber) Probe(url *url.URL, headers http.Header, timeout time.Duration) (api.Result, string, error) {
	client := &http.Client{
		Timeout:       timeout,
		Transport:     pr.transport,
		CheckRedirect: redirectChecker(pr.followNonLocalRedirects),
	}
	return DoHTTPGetProbe(url, headers, client)
}

// DoHTTPGetProbe checks if a GET request to the url succeeds.
// If the HTTP response code is successful (i.e. 400 > code >= 200), it returns Success.
// If the HTTP response code is unsuccessful or HTTP communication fails, it returns Failure.
// This is exported because some other packages may want to do direct HTTP probes.
func DoHTTPGetProbe(url *url.URL, headers http.Header, client HTTPInterface) (api.Result, string, error) {
	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		// Convert errors into failures to catch timeouts.
		return api.Failure, err.Error(), nil
	}
	return doHTTPProbe(req, url, headers, client)
}
