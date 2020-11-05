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

package analytics

import (
	"os"

	"kmodules.xyz/client-go/meta"
	"kmodules.xyz/client-go/tools/clusterid"

	"gomodules.xyz/x/analytics"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	Key = "APPSCODE_ANALYTICS_CLIENT_ID"
)

func ClientID() string {
	if id, found := os.LookupEnv(Key); found {
		return id
	}

	defer runtime.HandleCrash()

	if !meta.PossiblyInCluster() {
		return analytics.ClientID()
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return "$k8s$inclusterconfig"
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "$k8s$newforconfig"
	}
	id, err := clusterid.ClusterUID(client.CoreV1().Namespaces())
	if err != nil {
		return reasonForError(err)
	}
	return id
}

func reasonForError(err error) string {
	switch t := err.(type) {
	case kerr.APIStatus:
		return "$k8s$err$" + string(t.Status().Reason)
	}
	return "$k8s$err$" + trim(err.Error(), 32) // 32 = length of uuid
}

func trim(s string, length int) string {
	if len(s) > length {
		return s[:length]
	}
	return s
}
