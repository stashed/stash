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
	"context"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func IPv6EnabledInCluster(kc kubernetes.Interface) (bool, error) {
	svc, err := kc.CoreV1().Services(metav1.NamespaceDefault).Get(context.TODO(), "kubernetes", metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	clusterIPs := svc.Spec.ClusterIPs
	if len(clusterIPs) == 0 {
		clusterIPs = []string{svc.Spec.ClusterIP}
	}
	for _, ip := range clusterIPs {
		if strings.ContainsRune(ip, ':') {
			return true, nil
		}
	}
	return false, nil
}

func IPv6EnabledInKernel() (bool, error) {
	content, err := os.ReadFile("/sys/module/ipv6/parameters/disable")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(content)) == "0", nil
}
