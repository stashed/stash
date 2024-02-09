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

package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"kmodules.xyz/client-go/apiextensions"
	"kmodules.xyz/custom-resources/crds"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (_ AppBinding) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crds.MustCustomResourceDefinition(SchemeGroupVersion.WithResource(ResourceApps))
}

func (a AppBinding) URL() (string, error) {
	c := a.Spec.ClientConfig
	if c.URL != nil {
		return *c.URL, nil
	} else if c.Service != nil {
		ns := a.Namespace
		if c.Service.Namespace != "" {
			ns = c.Service.Namespace
		}
		u := url.URL{
			Scheme:   c.Service.Scheme,
			Host:     fmt.Sprintf("%s.%s.svc:%d", c.Service.Name, ns, c.Service.Port),
			Path:     c.Service.Path,
			RawQuery: c.Service.Query,
		}
		return u.String(), nil
	}
	return "", errors.New("connection url is missing")
}

func (a AppBinding) URLTemplate() (string, error) {
	rawurl, err := a.URL()
	if err != nil {
		return "", err
	}
	auth := fmt.Sprintf("{{%s}}:{{%s}}@", core.BasicAuthUsernameKey, core.BasicAuthPasswordKey)

	i := strings.Index(rawurl, "://")
	if i < 0 {
		return auth + rawurl, nil
	}
	return fmt.Sprintf(rawurl[:i+3] + auth + rawurl[i+3:]), nil
}

func (a AppBinding) Host() (string, error) {
	c := a.Spec.ClientConfig
	if c.Service != nil { // preferred source for MYSQL app binding
		ns := a.Namespace
		if c.Service.Namespace != "" {
			ns = c.Service.Namespace
		}
		return fmt.Sprintf("%s.%s.svc:%d", c.Service.Name, ns, c.Service.Port), nil
	} else if c.URL != nil {
		u, err := url.Parse(*c.URL)
		if err != nil {
			return ParseMySQLHost(*c.URL)
		}
		return u.Host, nil
	}
	return "", errors.New("connection url is missing")
}

func (a AppBinding) Hostname() (string, error) {
	c := a.Spec.ClientConfig
	if c.Service != nil { // preferred source for MYSQL app binding
		ns := a.Namespace
		if c.Service.Namespace != "" {
			ns = c.Service.Namespace
		}
		return fmt.Sprintf("%s.%s.svc", c.Service.Name, ns), nil
	} else if c.URL != nil {
		u, err := url.Parse(*c.URL)
		if err != nil {
			return ParseMySQLHostname(*c.URL)
		}
		return u.Hostname(), nil
	}
	return "", errors.New("connection url is missing")
}

func (a AppBinding) Port() (int32, error) {
	c := a.Spec.ClientConfig
	if c.Service != nil { // preferred source for MYSQL app binding
		return c.Service.Port, nil
	} else if c.URL != nil {
		var port string
		u, err := url.Parse(*c.URL)
		if err != nil {
			port, err = ParseMySQLPort(*c.URL)
			if err != nil {
				return 0, nil
			}
		} else {
			port = u.Port()
		}
		result, err := strconv.ParseInt(port, 10, 32)
		return int32(result), err
	}
	return 0, errors.New("connection url is missing")
}

func (a AppBinding) AppGroupResource() (string, string) {
	t := string(a.Spec.Type)
	idx := strings.LastIndexByte(t, '/')
	if idx == -1 {
		return "", t
	}
	return t[:idx], t[idx+1:]
}

// xref: https://github.com/kubernetes-sigs/service-catalog/blob/a204c0d26c60b42121aa608c39a179680e499d2a/pkg/controller/controller_binding.go#L605
func (a AppBinding) TransformSecret(kc kubernetes.Interface, credentials map[string][]byte) error {
	for _, t := range a.Spec.SecretTransforms {
		switch {
		case t.AddKey != nil:
			var value []byte
			if t.AddKey.StringValue != nil {
				value = []byte(*t.AddKey.StringValue)
			} else {
				value = t.AddKey.Value
			}
			credentials[t.AddKey.Key] = value
		case t.RenameKey != nil:
			value, ok := credentials[t.RenameKey.From]
			if ok {
				credentials[t.RenameKey.To] = value
				delete(credentials, t.RenameKey.From)
			}
		case t.AddKeysFrom != nil:
			secret, err := kc.CoreV1().
				Secrets(a.Namespace).
				Get(context.Background(), t.AddKeysFrom.SecretRef.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			for k, v := range secret.Data {
				credentials[k] = v
			}
		case t.RemoveKey != nil:
			delete(credentials, t.RemoveKey.Key)
		}
	}
	return nil
}
