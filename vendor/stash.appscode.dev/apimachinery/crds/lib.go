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

package crds

import (
	"embed"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"kmodules.xyz/client-go/apiextensions"
	"sigs.k8s.io/yaml"
)

//go:embed *.yaml
var fs embed.FS

func load(filename string, o any) error {
	data, err := fs.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, o)
}

func CustomResourceDefinition(gvr schema.GroupVersionResource) (*apiextensions.CustomResourceDefinition, error) {
	var out apiextensions.CustomResourceDefinition

	v1file := fmt.Sprintf("%s_%s.yaml", gvr.Group, gvr.Resource)
	if err := load(v1file, &out.V1); err != nil {
		return nil, err
	}

	if out.V1 == nil {
		return nil, fmt.Errorf("missing crd yamls for gvr: %s", gvr)
	}

	return &out, nil
}

func MustCustomResourceDefinition(gvr schema.GroupVersionResource) *apiextensions.CustomResourceDefinition {
	out, err := CustomResourceDefinition(gvr)
	if err != nil {
		panic(err)
	}
	return out
}
