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

package shared

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"text/template"

	kmapi "kmodules.xyz/client-go/api/v1"
	"kmodules.xyz/go-containerregistry/name"

	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"
)

const (
	GraphQueryVarSource      = "src"
	GraphQueryVarTargetGroup = "targetGroup"
	GraphQueryVarTargetKind  = "targetKind"
)

var pool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func (r ResourceLocator) GraphQuery(oid kmapi.OID) (string, map[string]interface{}, error) {
	if r.Query.Type == GraphQLQuery {
		vars := map[string]interface{}{
			GraphQueryVarSource:      string(oid),
			GraphQueryVarTargetGroup: r.Ref.Group,
			GraphQueryVarTargetKind:  r.Ref.Kind,
		}

		if r.Query.Raw != "" {
			return r.Query.Raw, vars, nil
		}
		return fmt.Sprintf(`query Find($src: String!, $targetGroup: String!, $targetKind: String!) {
  find(oid: $src) {
    refs: %s(group: $targetGroup, kind: $targetKind) {
      namespace
      name
    }
  }
}`, r.Query.ByLabel), vars, nil
	} else if r.Query.Type == RESTQuery {
		if r.Query.Raw == "" || !strings.Contains(r.Query.Raw, "{{") {
			return r.Query.Raw, nil, nil
		}

		tpl, err := template.New("").Funcs(sprig.TxtFuncMap()).Parse(r.Query.Raw)
		if err != nil {
			return "", nil, errors.Wrap(err, "failed to parse raw query")
		}
		// Do nothing and continue execution.
		// If printed, the result of the index operation is the string "<no value>".
		// We mitigate that later.
		tpl.Option("missingkey=default")

		objID, err := kmapi.ObjectIDMap(oid)
		if err != nil {
			return "", nil, errors.Wrapf(err, "failed to parse oid=%s", oid)
		}

		buf := pool.Get().(*bytes.Buffer)
		defer pool.Put(buf)
		buf.Reset()

		err = tpl.Execute(buf, objID)
		if err != nil {
			return "", nil, errors.Wrap(err, "failed to resolve template")
		}
		return buf.String(), nil, nil
	}
	return "", nil, fmt.Errorf("unknown query type %+v, oid %s", r, oid)
}

func (r ImageRegistrySpec) DockerHubProxy() string {
	addr := r.Proxies.DockerHub
	addr = strings.TrimSpace(addr)
	addr = strings.TrimSuffix(addr, "/")
	return addr
}

func (r ImageRegistrySpec) DockerLibraryProxy() string {
	addr := r.Proxies.DockerLibrary
	if addr == "" {
		addr = r.Proxies.DockerHub
	}
	addr = strings.TrimSpace(addr)
	addr = strings.TrimSuffix(addr, "/")
	return addr
}

func (r ImageRegistrySpec) GHCRProxy() string {
	addr := r.Proxies.GHCR
	addr = strings.TrimSpace(addr)
	addr = strings.TrimSuffix(addr, "/")
	return addr
}

func (r ImageRegistrySpec) QuayProxy() string {
	addr := r.Proxies.Quay
	addr = strings.TrimSpace(addr)
	addr = strings.TrimSuffix(addr, "/")
	return addr
}

func (r ImageRegistrySpec) KubernetesRegistryProxy() string {
	addr := r.Proxies.Kubernetes
	addr = strings.TrimSpace(addr)
	addr = strings.TrimSuffix(addr, "/")
	return addr
}

func (r ImageRegistrySpec) AppsCodeRegistryProxy() string {
	addr := r.Proxies.AppsCode
	addr = strings.TrimSpace(addr)
	addr = strings.TrimSuffix(addr, "/")
	return addr
}

const defaultTag = "latest"

func NewRef(spec ImageRegistrySpec, img string) (string, error) {
	ref, err := name.ParseReference(img)
	if err != nil {
		return "", err
	}

	// https://github.com/kmodules/go-containerregistry/blob/master/name/lib_test.go
	switch ref.Registry {
	case "index.docker.io":
		var result string
		_, bin, found := strings.Cut(ref.Repository, "library/")
		if found {
			addr := spec.DockerLibraryProxy()
			if addr != "" {
				result = addr + "/" + bin
			} else {
				result = bin
			}
		} else {
			addr := spec.DockerHubProxy()
			if addr != "" {
				result = addr + "/" + ref.Repository
			} else {
				result = ref.Repository
			}
		}
		if ref.Tag != "" && ref.Tag != defaultTag {
			result += ":" + ref.Tag
		}
		return result, nil
	case "ghcr.io":
		result := spec.GHCRProxy() + "/" + ref.Repository
		if ref.Tag != "" && ref.Tag != defaultTag {
			result += ":" + ref.Tag
		}
		return result, nil
	case "quay.io":
		result := spec.QuayProxy() + "/" + ref.Repository
		if ref.Tag != "" && ref.Tag != defaultTag {
			result += ":" + ref.Tag
		}
		return result, nil
	case "registry.k8s.io":
		result := spec.KubernetesRegistryProxy() + "/" + ref.Repository
		if ref.Tag != "" && ref.Tag != defaultTag {
			result += ":" + ref.Tag
		}
		return result, nil
	case "r.appscode.com":
		result := spec.AppsCodeRegistryProxy() + "/" + ref.Repository
		if ref.Tag != "" && ref.Tag != defaultTag {
			result += ":" + ref.Tag
		}
		return result, nil
	}
	return "", fmt.Errorf("registry not support for image %s", img)
}

func MustNewRef(spec ImageRegistrySpec, img string) string {
	out, err := NewRef(spec, img)
	if err != nil {
		panic(err)
	}
	return out
}
