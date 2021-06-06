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

package resourceclasses

import (
	"embed"
	"strings"

	"kmodules.xyz/resource-metadata/apis/meta/v1alpha1"

	"github.com/gobuffalo/flect"
	"golang.org/x/net/publicsuffix"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

//go:embed *.yaml
var fs embed.FS

func FS() embed.FS {
	return fs
}

func ResourceClassName(apiGroup string) string {
	name := apiGroup
	name = strings.TrimSuffix(name, ".k8s.io")
	name = strings.TrimSuffix(name, ".x-k8s.io")

	idx := strings.IndexRune(name, '.')
	if idx >= 0 {
		eTLD, icann := publicsuffix.PublicSuffix(name)
		if icann {
			name = strings.TrimSuffix(name, "."+eTLD)
		}
		parts := strings.Split(name, ".")
		for i := 0; i < len(parts)/2; i++ {
			j := len(parts) - i - 1
			parts[i], parts[j] = parts[j], parts[i]
		}
		name = strings.Join(parts, " ")
	}
	if name != "" {
		name = flect.Titleize(flect.Humanize(flect.Singularize(name)))
	} else {
		name = "Core"
	}
	return name
}

func LoadByGVR(gvr schema.GroupVersionResource) (*v1alpha1.ResourceClass, error) {
	name := ResourceClassName(gvr.Group)
	return LoadByName(name)
}

func LoadByName(name string) (*v1alpha1.ResourceClass, error) {
	filename := strings.ToLower(name) + ".yaml"
	return LoadByFile(filename)
}

func LoadByFile(filename string) (*v1alpha1.ResourceClass, error) {
	data, err := fs.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var obj v1alpha1.ResourceClass
	err = yaml.Unmarshal(data, &obj)
	if err != nil {
		return nil, err
	}
	return &obj, nil
}
