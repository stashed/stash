/*
Copyright The Kmodules Authors.

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
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"kmodules.xyz/custom-resources/api/crds"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func (_ AppBinding) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crds.MustCustomResourceDefinition(SchemeGroupVersion.WithResource(ResourceApps))
}

func (a AppBinding) URL() (string, error) {
	c := a.Spec.ClientConfig
	if c.URL != nil {
		return *c.URL, nil
	} else if c.Service != nil {
		u := url.URL{
			Scheme:   c.Service.Scheme,
			Host:     fmt.Sprintf("%s.%s.svc:%d", c.Service.Name, a.Namespace, c.Service.Port),
			Path:     c.Service.Path,
			RawQuery: c.Service.Query,
		}
		return u.String(), nil
	}
	return "", errors.New("connection url is missing")
}

const (
	KeyUsername = "username"
	KeyPassword = "password"
)

func (a AppBinding) URLTemplate() (string, error) {
	rawurl, err := a.URL()
	if err != nil {
		return "", err
	}
	auth := fmt.Sprintf("{{%s}}:{{%s}}@", KeyUsername, KeyPassword)

	i := strings.Index(rawurl, "://")
	if i < 0 {
		return auth + rawurl, nil
	}
	return fmt.Sprintf(rawurl[:i+3] + auth + rawurl[i+3:]), nil
}

func (a AppBinding) Host() (string, error) {
	c := a.Spec.ClientConfig
	if c.Service != nil { // preferred source for MYSQL app binding
		return fmt.Sprintf("%s.%s.svc:%d", c.Service.Name, a.Namespace, c.Service.Port), nil
	} else if c.URL != nil {
		u, err := url.Parse(*c.URL)
		if err != nil {
			return "", err
		}
		return u.Host, nil
	}
	return "", errors.New("connection url is missing")
}

func (a AppBinding) Hostname() (string, error) {
	c := a.Spec.ClientConfig
	if c.Service != nil { // preferred source for MYSQL app binding
		return fmt.Sprintf("%s.%s.svc", c.Service.Name, a.Namespace), nil
	} else if c.URL != nil {
		u, err := url.Parse(*c.URL)
		if err != nil {
			return "", err
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
		u, err := url.Parse(*c.URL)
		if err != nil {
			return 0, err
		}
		port, err := strconv.ParseInt(u.Port(), 10, 32)
		return int32(port), err
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
