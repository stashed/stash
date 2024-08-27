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

package meta

import (
	"net"
	"os"
	"strings"

	core "k8s.io/api/core/v1"
)

// xref: https://kubernetes.io/docs/concepts/workloads/pods/downward-api/
// xref: https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/#use-pod-fields-as-values-for-environment-variables

func NodeName() string {
	return os.Getenv("NODE_NAME")
}

func PodName() string {
	if name := os.Getenv("POD_NAME"); name != "" {
		return name
	}
	s, _ := os.Hostname()
	return s
}

func PodNamespace() string {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}

	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return core.NamespaceDefault
}

func PodServiceAccount() string {
	return os.Getenv("POD_SERVICE_ACCOUNT")
}

// PossiblyInCluster returns true if loading an inside-kubernetes-cluster is possible.
// ref: https://github.com/kubernetes/kubernetes/blob/v1.18.3/staging/src/k8s.io/client-go/tools/clientcmd/client_config.go#L537
func PossiblyInCluster() bool {
	fi, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token")
	return os.Getenv("KUBERNETES_SERVICE_HOST") != "" &&
		os.Getenv("KUBERNETES_SERVICE_PORT") != "" &&
		err == nil && !fi.IsDir()
}

func ClusterDomain() string {
	defaultDomain := func() string {
		if v, ok := os.LookupEnv("KUBE_CLUSTER_DOMAIN"); ok {
			return v
		}
		return "cluster.local"
	}

	if !PossiblyInCluster() {
		return defaultDomain()
	}

	const k8sService = "kubernetes.default.svc"
	domain, err := net.LookupCNAME(k8sService)
	if err != nil {
		return defaultDomain()
	}
	return strings.Trim(strings.TrimPrefix(domain, k8sService), ".")
}
