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

package crds

import (
	"fmt"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

func CustomResourceDefinition(gvr schema.GroupVersionResource) (*apiextensions.CustomResourceDefinition, error) {
	data, err := Asset(fmt.Sprintf("%s_%s.yaml", gvr.Group, gvr.Resource))
	if err != nil {
		return nil, err
	}
	var out apiextensions.CustomResourceDefinition
	err = yaml.Unmarshal(data, &out)
	if err != nil {
		return nil, err
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
