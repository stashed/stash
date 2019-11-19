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

package probe

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	api "kmodules.xyz/prober/api"
	api_v1 "kmodules.xyz/prober/api/v1"
	execprobe "kmodules.xyz/prober/probe/exec"
	httpprobe "kmodules.xyz/prober/probe/http"
	tcpprobe "kmodules.xyz/prober/probe/tcp"

	"github.com/appscode/go/log"
	core "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Prober struct {
	HttpGet  httpprobe.GetProber
	HttpPost httpprobe.PostProber
	Tcp      tcpprobe.Prober
	Exec     execprobe.Prober
	Config   *rest.Config
}

// NewProber creates a Prober instance that can be used to run httpGet, httpPost, tcp or exec probe.
func NewProber(config *rest.Config) *Prober {
	const followNonLocalRedirects = false

	return &Prober{
		HttpGet:  httpprobe.NewHttpGet(followNonLocalRedirects),
		HttpPost: httpprobe.NewHttpPost(followNonLocalRedirects),
		Tcp:      tcpprobe.New(),
		Exec:     execprobe.New(),
		Config:   config,
	}
}

func RunProbe(config *rest.Config, probes *api_v1.Handler, podName, namespace string) error {
	prober := NewProber(config)

	var pod *core.Pod
	if podName != "" {
		kubeClient, err := kubernetes.NewForConfig(config)
		if err != nil {
			return fmt.Errorf("failed to create kuberentes client. Error: %v", err.Error())
		}

		pod, err = kubeClient.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("filed to get pod %s/%s. Error: %v", namespace, podName, err.Error())
		}
	}

	return prober.executeProbe(probes, pod, api.DefaultProbeTimeout)
}

func (pb *Prober) executeProbe(p *api_v1.Handler, pod *core.Pod, timeout time.Duration) error {
	if p.Exec != nil {
		log.Debugf("Exec-Probe Pod: %v, Container: %v, Command: %v", formatPod(pod), p.ContainerName, p.Exec.Command)
		res, resp, err := pb.Exec.Probe(pb.Config, pod, p.ContainerName, p.Exec.Command)
		if res != api.Success && res != api.Warning {
			return handleProbeFailure("exec", res, resp, err)
		}
	}
	if p.HTTPGet != nil {
		res, resp, err := pb.executeHttpGet(p, pod, timeout)
		if res != api.Success && res != api.Warning {
			return handleProbeFailure("httpGet", res, resp, err)
		}
	}
	if p.HTTPPost != nil {
		res, resp, err := pb.executeHttpPost(p, pod, timeout)
		if res != api.Success && res != api.Warning {
			return handleProbeFailure("httpPost", res, resp, err)
		}
	}
	if p.TCPSocket != nil {
		res, resp, err := pb.executeTcpProbe(p, pod, timeout)
		if res != api.Success && res != api.Warning {
			return handleProbeFailure("tcp", res, resp, err)
		}
	}
	return nil
}

func (pb *Prober) executeHttpGet(p *api_v1.Handler, pod *core.Pod, timeout time.Duration) (api.Result, string, error) {
	scheme := strings.ToLower(string(p.HTTPGet.Scheme))
	host := p.HTTPGet.Host
	if host == "" {
		host = pod.Status.PodIP
	}
	port, err := extractPort(p.HTTPGet.Port, pod, p.ContainerName)
	if err != nil {
		return api.Unknown, "", err
	}
	path := p.HTTPGet.Path
	log.Debugf("HTTP-Probe Host: %v://%v, Port: %v, Path: %v", scheme, host, port, path)
	targetURL := formatURL(scheme, host, port, path)
	headers := buildHeader(p.HTTPGet.HTTPHeaders)
	log.Debugf("HTTP-Probe Headers: %v", headers)
	return pb.HttpGet.Probe(targetURL, headers, timeout)
}

func (pb *Prober) executeHttpPost(p *api_v1.Handler, pod *core.Pod, timeout time.Duration) (api.Result, string, error) {
	scheme := strings.ToLower(string(p.HTTPPost.Scheme))
	host := p.HTTPPost.Host
	if host == "" {
		host = pod.Status.PodIP
	}
	port, err := extractPort(p.HTTPPost.Port, pod, p.ContainerName)
	if err != nil {
		return api.Unknown, "", err
	}
	path := p.HTTPPost.Path
	log.Debugf("HTTP-Probe Host: %v://%v, Port: %v, Path: %v", scheme, host, port, path)
	targetURL := formatURL(scheme, host, port, path)
	headers := buildHeader(p.HTTPPost.HTTPHeaders)
	log.Debugf("HTTP-Probe Headers: %v", headers)
	return pb.HttpPost.Probe(targetURL, headers, toValues(p.HTTPPost.Form), p.HTTPPost.Body, timeout)
}

func (pb *Prober) executeTcpProbe(p *api_v1.Handler, pod *core.Pod, timeout time.Duration) (api.Result, string, error) {
	port, err := extractPort(p.TCPSocket.Port, pod, p.ContainerName)
	if err != nil {
		return api.Unknown, "", err
	}
	host := p.TCPSocket.Host
	if host == "" {
		host = pod.Status.PodIP
	}
	log.Debugf("TCP-Probe Host: %v, Port: %v, Timeout: %v", host, port, timeout)
	return pb.Tcp.Probe(host, port, timeout)
}

func toValues(formEntry []api_v1.FormEntry) url.Values {
	if len(formEntry) == 0 {
		return nil
	}
	out := url.Values{}
	for _, v := range formEntry {
		out[v.Key] = v.Values
	}
	return out
}

// buildHeaderMap takes a list of HTTPHeader <name, value> string
// pairs and returns a populated string->[]string http.Header map.
func buildHeader(headerList []v1.HTTPHeader) http.Header {
	headers := make(http.Header)
	for _, header := range headerList {
		headers[header.Name] = append(headers[header.Name], header.Value)
	}
	return headers
}

func extractPort(param intstr.IntOrString, pod *core.Pod, containerName string) (int, error) {
	port := -1
	var err error

	switch param.Type {
	case intstr.Int:
		port = param.IntValue()
	case intstr.String:
		if pod == nil {
			return port, fmt.Errorf("failed to extract port. invalid pod")
		}

		var container core.Container
		found := false
		for i := range pod.Spec.Containers {
			if pod.Spec.Containers[i].Name == containerName {
				container = pod.Spec.Containers[i]
				found = true
				break
			}
		}
		if !found {
			return port, fmt.Errorf("failed to extract port. container not found")
		}
		if port, err = findPortByName(container, param.StrVal); err != nil {
			// Last ditch effort - maybe it was an int stored as string?
			if port, err = strconv.Atoi(param.StrVal); err != nil {
				return port, err
			}
		}
	default:
		return port, fmt.Errorf("intOrString had no kind: %+v", param)
	}

	if port > 0 && port < 65536 {
		return port, nil
	}
	return port, fmt.Errorf("invalid port number: %v", port)
}

func handleProbeFailure(probeType string, result api.Result, resp string, probeErr error) error {
	switch result {
	case api.Unknown:
		return fmt.Errorf("failed to execute %q probe. Error: %v", probeType, probeErr)
	case api.Failure:
		return fmt.Errorf("failed to execute %q probe. Error: %v. Response: %s", probeType, probeErr, resp)
	}
	return nil
}

// findPortByName is a helper function to look up a port in a container by name.
func findPortByName(container core.Container, portName string) (int, error) {
	for _, port := range container.Ports {
		if port.Name == portName {
			return int(port.ContainerPort), nil
		}
	}
	return 0, fmt.Errorf("port %s not found", portName)
}

// formatURL formats a URL from args.  For testability.
func formatURL(scheme string, host string, port int, path string) *url.URL {
	u, err := url.Parse(path)
	// Something is busted with the path, but it's too late to reject it. Pass it along as is.
	if err != nil {
		u = &url.URL{
			Path: path,
		}
	}
	u.Scheme = scheme
	u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	return u
}

// formatPod returns a string representing a pod in a consistent human readable format,
// with pod UID as part of the string.
func formatPod(pod *v1.Pod) string {
	return podDesc(pod.Name, pod.Namespace, pod.UID)
}

// podDesc returns a string representing a pod in a consistent human readable format,
// with pod UID as part of the string.
func podDesc(podName, podNamespace string, podUID types.UID) string {
	// Use underscore as the delimiter because it is not allowed in pod name
	// (DNS subdomain format), while allowed in the container name format.
	return fmt.Sprintf("%s_%s(%s)", podName, podNamespace, podUID)
}
