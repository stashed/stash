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

package resourcedescriptors

import (
	"embed"
	"fmt"
	"strings"

	"kmodules.xyz/resource-metadata/apis/meta/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

//go:embed **/**/*.yaml
var fs embed.FS

func FS() embed.FS {
	return fs
}

func LoadByGVR(gvr schema.GroupVersionResource) (*v1alpha1.ResourceDescriptor, error) {
	var filename string
	if gvr.Group == "" && gvr.Version == "v1" {
		filename = fmt.Sprintf("core/v1/%s.yaml", gvr.Resource)
	} else {
		filename = fmt.Sprintf("%s/%s/%s.yaml", gvr.Group, gvr.Version, gvr.Resource)
	}
	return LoadByFile(filename)
}

func LoadByName(name string) (*v1alpha1.ResourceDescriptor, error) {
	filename := strings.Replace(name, "-", "/", 2) + ".yaml"
	return LoadByFile(filename)
}

func LoadByFile(filename string) (*v1alpha1.ResourceDescriptor, error) {
	data, err := fs.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var obj v1alpha1.ResourceDescriptor
	err = yaml.Unmarshal(data, &obj)
	if err != nil {
		return nil, err
	}
	return &obj, nil
}
