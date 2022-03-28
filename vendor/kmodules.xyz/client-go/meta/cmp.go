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

package meta

import (
	"github.com/google/go-cmp/cmp"
	jsoniter "github.com/json-iterator/go"
	jsondiff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var cmpOptions = []cmp.Option{
	cmp.Comparer(func(x, y resource.Quantity) bool {
		return x.Cmp(y) == 0
	}),
	cmp.Comparer(func(x, y *metav1.Time) bool {
		if x == nil && y == nil {
			return true
		}
		if x != nil && y != nil {
			return x.Time.Equal(y.Time)
		}
		return false
	}),
}

func Diff(x, y interface{}) string {
	return cmp.Diff(x, y, cmpOptions...)
}

func Equal(x, y interface{}) bool {
	return cmp.Equal(x, y, cmpOptions...)
}

const LastAppliedConfigAnnotation = "kubectl.kubernetes.io/last-applied-configuration"

// EqualAnnotation checks equality of annotations skipping `kubectl.kubernetes.io/last-applied-configuration` key
func EqualAnnotation(x, y map[string]string) bool {
	xLen := len(x)
	if _, found := x[LastAppliedConfigAnnotation]; found {
		xLen--
	}
	yLen := len(y)
	if _, found := y[LastAppliedConfigAnnotation]; found {
		yLen--
	}
	if xLen != yLen {
		return false
	}

	for k, v := range x {
		if k == LastAppliedConfigAnnotation {
			continue
		}
		if y[k] != v {
			return false
		}
	}
	return true
}

func JsonDiff(old, new interface{}) (string, error) {
	json := jsoniter.ConfigFastest
	oldBytes, err := json.Marshal(old)
	if err != nil {
		return "", err
	}

	newBytes, err := json.Marshal(new)
	if err != nil {
		return "", err
	}

	// Then, compare them
	differ := jsondiff.New()
	d, err := differ.Compare(oldBytes, newBytes)
	if err != nil {
		return "", err
	}

	var aJson map[string]interface{}
	if err := json.Unmarshal(oldBytes, &aJson); err != nil {
		return "", err
	}

	format := formatter.NewAsciiFormatter(aJson, formatter.AsciiFormatterConfig{
		ShowArrayIndex: true,
		Coloring:       false,
	})
	return format.Format(d)
}
