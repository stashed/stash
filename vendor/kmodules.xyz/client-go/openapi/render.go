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

package openapi

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/go-openapi/spec"
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/kube-openapi/pkg/builder"
	"k8s.io/kube-openapi/pkg/common"
)

type TypeInfo struct {
	GroupVersion    schema.GroupVersion
	Resource        string
	Kind            string
	NamespaceScoped bool
}

type VersionResource struct {
	Version  string
	Resource string
}

type Config struct {
	Scheme *runtime.Scheme
	Codecs serializer.CodecFactory

	Info               spec.InfoProps
	OpenAPIDefinitions []common.GetOpenAPIDefinitions
	Resources          []TypeInfo
	GetterResources    []TypeInfo
	ListerResources    []TypeInfo
	CDResources        []TypeInfo
	RDResources        []TypeInfo
}

func (c *Config) GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	out := map[string]common.OpenAPIDefinition{}
	for _, def := range c.OpenAPIDefinitions {
		for k, v := range def(ref) {
			out[k] = v
		}
	}
	return out
}

func RenderOpenAPISpec(cfg Config) (string, error) {
	// we need to add the options to empty v1
	// TODO fix the server code to avoid this
	metav1.AddToGroupVersion(cfg.Scheme, schema.GroupVersion{Version: "v1"})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	cfg.Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)

	recommendedOptions := genericoptions.NewRecommendedOptions("/registry/foo.com", cfg.Codecs.LegacyCodec(), &genericoptions.ProcessInfo{})
	recommendedOptions.SecureServing.BindPort = 8443
	recommendedOptions.Etcd = nil
	recommendedOptions.Authentication = nil
	recommendedOptions.Authorization = nil
	recommendedOptions.CoreAPI = nil
	recommendedOptions.Admission = nil

	// TODO have a "real" external address
	if err := recommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		glog.Fatal(fmt.Errorf("error creating self-signed certificates: %v", err))
	}

	serverConfig := genericapiserver.NewRecommendedConfig(cfg.Codecs)
	if err := recommendedOptions.ApplyTo(serverConfig); err != nil {
		return "", err
	}
	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(cfg.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(cfg.Scheme))
	serverConfig.OpenAPIConfig.Info.InfoProps = cfg.Info

	genericServer, err := serverConfig.Complete().New("stash-server", genericapiserver.NewEmptyDelegate()) // completion is done in Complete, no need for a second time
	if err != nil {
		return "", err
	}

	table := map[string]map[VersionResource]rest.Storage{}
	{
		for _, ti := range cfg.Resources {
			var resmap map[VersionResource]rest.Storage
			if m, found := table[ti.GroupVersion.Group]; found {
				resmap = m
			} else {
				resmap = map[VersionResource]rest.Storage{}
				table[ti.GroupVersion.Group] = resmap
			}

			gvk := ti.GroupVersion.WithKind(ti.Kind)
			obj, err := cfg.Scheme.New(gvk)
			if err != nil {
				return "", err
			}
			list, err := cfg.Scheme.New(ti.GroupVersion.WithKind(ti.Kind + "List"))
			if err != nil {
				return "", err
			}

			resmap[VersionResource{Version: ti.GroupVersion.Version, Resource: ti.Resource}] = NewStandardStorage(ResourceInfo{
				gvk:             gvk,
				obj:             obj,
				list:            list,
				namespaceScoped: ti.NamespaceScoped,
			})
		}
	}
	{
		for _, ti := range cfg.GetterResources {
			var resmap map[VersionResource]rest.Storage
			if m, found := table[ti.GroupVersion.Group]; found {
				resmap = m
			} else {
				resmap = map[VersionResource]rest.Storage{}
				table[ti.GroupVersion.Group] = resmap
			}

			gvk := ti.GroupVersion.WithKind(ti.Kind)
			obj, err := cfg.Scheme.New(gvk)
			if err != nil {
				return "", err
			}

			resmap[VersionResource{Version: ti.GroupVersion.Version, Resource: ti.Resource}] = NewGetterStorage(ResourceInfo{
				gvk:             gvk,
				obj:             obj,
				namespaceScoped: ti.NamespaceScoped,
			})
		}
	}
	{
		for _, ti := range cfg.ListerResources {
			var resmap map[VersionResource]rest.Storage
			if m, found := table[ti.GroupVersion.Group]; found {
				resmap = m
			} else {
				resmap = map[VersionResource]rest.Storage{}
				table[ti.GroupVersion.Group] = resmap
			}

			gvk := ti.GroupVersion.WithKind(ti.Kind)
			obj, err := cfg.Scheme.New(gvk)
			if err != nil {
				return "", err
			}
			list, err := cfg.Scheme.New(ti.GroupVersion.WithKind(ti.Kind + "List"))
			if err != nil {
				return "", err
			}

			resmap[VersionResource{Version: ti.GroupVersion.Version, Resource: ti.Resource}] = NewListerStorage(ResourceInfo{
				gvk:             gvk,
				obj:             obj,
				list:            list,
				namespaceScoped: ti.NamespaceScoped,
			})
		}
	}
	{
		for _, ti := range cfg.CDResources {
			var resmap map[VersionResource]rest.Storage
			if m, found := table[ti.GroupVersion.Group]; found {
				resmap = m
			} else {
				resmap = map[VersionResource]rest.Storage{}
				table[ti.GroupVersion.Group] = resmap
			}

			gvk := ti.GroupVersion.WithKind(ti.Kind)
			obj, err := cfg.Scheme.New(gvk)
			if err != nil {
				return "", err
			}

			resmap[VersionResource{Version: ti.GroupVersion.Version, Resource: ti.Resource}] = NewCDStorage(ResourceInfo{
				gvk:             gvk,
				obj:             obj,
				namespaceScoped: ti.NamespaceScoped,
			})
		}
	}
	{
		for _, ti := range cfg.RDResources {
			var resmap map[VersionResource]rest.Storage
			if m, found := table[ti.GroupVersion.Group]; found {
				resmap = m
			} else {
				resmap = map[VersionResource]rest.Storage{}
				table[ti.GroupVersion.Group] = resmap
			}

			gvk := ti.GroupVersion.WithKind(ti.Kind)
			obj, err := cfg.Scheme.New(gvk)
			if err != nil {
				return "", err
			}
			list, err := cfg.Scheme.New(ti.GroupVersion.WithKind(ti.Kind + "List"))
			if err != nil {
				return "", err
			}

			resmap[VersionResource{Version: ti.GroupVersion.Version, Resource: ti.Resource}] = NewRDStorage(ResourceInfo{
				gvk:             gvk,
				obj:             obj,
				list:            list,
				namespaceScoped: ti.NamespaceScoped,
			})
		}
	}

	for group, resmap := range table {
		apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(group, cfg.Scheme, metav1.ParameterCodec, cfg.Codecs)
		for vr, s := range resmap {
			if _, found := apiGroupInfo.VersionedResourcesStorageMap[vr.Version]; !found {
				apiGroupInfo.VersionedResourcesStorageMap[vr.Version] = make(map[string]rest.Storage)
			}
			apiGroupInfo.VersionedResourcesStorageMap[vr.Version][vr.Resource] = s
		}
		if err := genericServer.InstallAPIGroup(&apiGroupInfo); err != nil {
			return "", err
		}
	}

	spec, err := builder.BuildOpenAPISpec(genericServer.Handler.GoRestfulContainer.RegisteredWebServices(), serverConfig.OpenAPIConfig)
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
