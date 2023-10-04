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

package clientcmd

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
)

// https://github.com/kubernetes/client-go/issues/711#issuecomment-730112049
func BuildKubeConfig(cfg *rest.Config, namespace string) (*clientcmdapi.Config, error) {
	if err := rest.LoadTLSFiles(cfg); err != nil {
		return nil, err
	}

	clusters := make(map[string]*clientcmdapi.Cluster)
	clusters["default-cluster"] = &clientcmdapi.Cluster{
		Server:                   cfg.Host,
		CertificateAuthorityData: cfg.CAData,
	}

	contexts := make(map[string]*clientcmdapi.Context)
	contexts["default-context"] = &clientcmdapi.Context{
		Cluster:   "default-cluster",
		Namespace: namespace,
		AuthInfo:  "default-user",
	}

	authinfos := make(map[string]*clientcmdapi.AuthInfo)
	authinfos["default-user"] = &clientcmdapi.AuthInfo{
		LocationOfOrigin:      "",
		ClientCertificate:     "",
		ClientCertificateData: cfg.CertData,
		ClientKey:             "",
		ClientKeyData:         cfg.KeyData,
		Token:                 cfg.BearerToken,
		TokenFile:             "",
		Impersonate:           cfg.Impersonate.UserName,
		ImpersonateUID:        cfg.Impersonate.UID,
		ImpersonateGroups:     cfg.Impersonate.Groups,
		ImpersonateUserExtra:  cfg.Impersonate.Extra,
		Username:              cfg.Username,
		Password:              cfg.Password,
		AuthProvider:          cfg.AuthProvider,
		Exec:                  cfg.ExecProvider,
		Extensions:            nil,
	}

	return &clientcmdapi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       clusters,
		Contexts:       contexts,
		CurrentContext: "default-context",
		AuthInfos:      authinfos,
	}, nil
}

func BuildKubeConfigBytes(cfg *rest.Config, namespace string) ([]byte, error) {
	clientConfig, err := BuildKubeConfig(cfg, namespace)
	if err != nil {
		return nil, err
	}
	return runtime.Encode(clientcmdlatest.Codec, clientConfig)
}
