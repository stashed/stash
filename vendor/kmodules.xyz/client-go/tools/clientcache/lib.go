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

package clientcache

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gregjones/httpcache"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
)

type enableResponseCaching struct {
	rt     http.RoundTripper
	maxAge time.Duration
}

func (rt *enableResponseCaching) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := rt.rt.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	resp.Header.Set("Cache-Control", fmt.Sprintf("max-age=%d", int(rt.maxAge.Seconds())))
	return resp, nil
}

func fnEnableResponseCaching(maxAge time.Duration) func(rt http.RoundTripper) http.RoundTripper {
	return func(rt http.RoundTripper) http.RoundTripper {
		return &enableResponseCaching{rt, maxAge}
	}
}

var _ http.RoundTripper = &enableResponseCaching{}

func fnCacheResponse(cache httpcache.Cache) func(rt http.RoundTripper) http.RoundTripper {
	return func(rt http.RoundTripper) http.RoundTripper {
		t := httpcache.NewTransport(cache)
		t.Transport = rt
		return t
	}
}

func ConfigFor(config *rest.Config, maxAge time.Duration, cache httpcache.Cache) *rest.Config {
	c2 := rest.CopyConfig(config)
	c2.Wrap(transport.Wrappers(fnEnableResponseCaching(maxAge), fnCacheResponse(cache)))
	return c2
}
