package http

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	api "kmodules.xyz/prober/api"

	"github.com/appscode/go/log"
	utilio "k8s.io/utils/io"
)

const (
	ContentType           = "Content-Type"
	ContentUrlEncodedForm = "application/x-www-form-urlencoded"
	ContentJson           = "application/json"
	ContentPlainText      = "text/plain"
)

// HTTPInterface is an interface for making HTTP requests, that returns a response and error.
type HTTPInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

func doHTTPProbe(req *http.Request, url *url.URL, headers http.Header, client HTTPInterface) (api.Result, string, error) {
	if _, ok := headers["User-Agent"]; !ok {
		if headers == nil {
			headers = http.Header{}
		}
		// explicitly set User-Agent so it's not set to default Go value
		headers.Set("User-Agent", "kmodules.xyz/client-go/release-11.0")
	}
	req.Header = headers
	if headers.Get("Host") != "" {
		req.Host = headers.Get("Host")
	}
	res, err := client.Do(req)
	if err != nil {
		// Convert errors into failures to catch timeouts.
		return api.Failure, err.Error(), nil
	}
	defer res.Body.Close()
	b, err := utilio.ReadAtMost(res.Body, maxRespBodyLength)
	if err != nil {
		if err == utilio.ErrLimitReached {
			log.Debugf("Non fatal body truncation for %s, Response: %v", url.String(), *res)
		} else {
			return api.Failure, "", err
		}
	}
	respBody := string(b)
	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusBadRequest {
		if res.StatusCode >= http.StatusMultipleChoices { // Redirect
			log.Debugf("Probe terminated redirects for %s, Response: %v", url.String(), *res)
			return api.Warning, respBody, nil
		}
		log.Debugf("Probe succeeded for %s, Response: %v", url.String(), *res)
		return api.Success, respBody, nil
	}
	log.Debugf("Probe failed for %s with request headers %v, response body: %v", url.String(), headers, respBody)
	return api.Failure, fmt.Sprintf("HTTP probe failed with statuscode: %d", res.StatusCode), nil
}

func redirectChecker(followNonLocalRedirects bool) func(*http.Request, []*http.Request) error {
	if followNonLocalRedirects {
		return nil // Use the default http client checker.
	}

	return func(req *http.Request, via []*http.Request) error {
		if req.URL.Hostname() != via[0].URL.Hostname() {
			return http.ErrUseLastResponse
		}
		// Default behavior: stop after 10 redirects.
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
}
