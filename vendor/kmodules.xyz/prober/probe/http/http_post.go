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
	"strings"
	"time"

	api "kmodules.xyz/prober/api"

	"github.com/gabriel-vasile/mimetype"
	utilnet "k8s.io/apimachinery/pkg/util/net"
)

// New creates PostProber that will skip TLS verification while probing.
// followNonLocalRedirects configures whether the prober should follow redirects to a different hostname.
//   If disabled, redirects to other hosts will trigger a warning result.
func NewHttpPost(followNonLocalRedirects bool) PostProber {
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	return NewPostWithTLSConfig(tlsConfig, followNonLocalRedirects)
}

// NewWithTLSConfig takes tls config as parameter.
// followNonLocalRedirects configures whether the prober should follow redirects to a different hostname.
//   If disabled, redirects to other hosts will trigger a warning result.
func NewPostWithTLSConfig(config *tls.Config, followNonLocalRedirects bool) PostProber {
	// We do not want the probe use node's local proxy set.
	transport := utilnet.SetTransportDefaults(
		&http.Transport{
			TLSClientConfig:   config,
			DisableKeepAlives: true,
			Proxy:             http.ProxyURL(nil),
		})
	return httpPostProber{transport, followNonLocalRedirects}
}

// PostProber is an interface that defines the Probe function for doing HTTP probe.
type PostProber interface {
	Probe(url *url.URL, headers http.Header, form url.Values, body string, timeout time.Duration) (api.Result, string, error)
}

type httpPostProber struct {
	transport               *http.Transport
	followNonLocalRedirects bool
}

// Probe returns a ProbeRunner capable of running an HTTP check.
func (pr httpPostProber) Probe(url *url.URL, headers http.Header, form url.Values, body string, timeout time.Duration) (api.Result, string, error) {
	client := &http.Client{
		Timeout:       timeout,
		Transport:     pr.transport,
		CheckRedirect: redirectChecker(pr.followNonLocalRedirects),
	}
	return DoHTTPPostProbe(url, headers, client, form, body)
}

// DoHTTPPostProbe checks if a POST request to the url succeeds.
// If the HTTP response code is successful (i.e. 400 > code >= 200), it returns Success.
// If the HTTP response code is unsuccessful or HTTP communication fails, it returns Failure.
// This is exported because some other packages may want to do direct HTTP probes.
func DoHTTPPostProbe(addr *url.URL, headers http.Header, client HTTPInterface, form url.Values, body string) (api.Result, string, error) {
	var req *http.Request
	var err error

	if headers == nil {
		headers = http.Header{}
	}

	if form != nil {
		req, err = http.NewRequest(http.MethodPost, addr.String(), strings.NewReader(form.Encode()))
		if err != nil {
			// Convert errors into failures to catch timeouts.
			return api.Failure, err.Error(), nil
		}
		headers.Set(ContentType, ContentUrlEncodedForm)
	} else if len(body) > 0 {
		req, err = http.NewRequest(http.MethodPost, addr.String(), strings.NewReader(body))
		if err != nil {
			// Convert errors into failures to catch timeouts.
			return api.Failure, err.Error(), nil
		}
		mime, _ := mimetype.Detect([]byte(body))
		headers.Set(ContentType, mime)
	} else {
		req, err = http.NewRequest(http.MethodPost, addr.String(), nil)
		if err != nil {
			// Convert errors into failures to catch timeouts.
			return api.Failure, err.Error(), nil
		}
	}

	return doHTTPProbe(req, addr, headers, client)
}
