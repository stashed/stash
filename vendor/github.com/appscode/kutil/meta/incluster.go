package meta

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	core "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

func Namespace() string {
	if ns := os.Getenv("KUBE_NAMESPACE"); ns != "" {
		return ns
	}
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return core.NamespaceDefault
}

// PossiblyInCluster returns true if loading an inside-kubernetes-cluster is possible.
func PossiblyInCluster() bool {
	fi, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token")
	return os.Getenv("KUBERNETES_SERVICE_HOST") != "" &&
		os.Getenv("KUBERNETES_SERVICE_PORT") != "" &&
		err == nil && !fi.IsDir()
}

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
			DualStack: true,
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
