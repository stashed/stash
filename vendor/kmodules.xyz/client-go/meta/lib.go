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
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
)

// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/#labels
// ref: https://github.com/kubernetes-sigs/application/blob/4ead7f1b87048b7717b3e474a21fdc07e6bce636/pkg/controller/application/application_controller.go#L28
const (
	NameLabelKey      = "app.kubernetes.io/name"
	VersionLabelKey   = "app.kubernetes.io/version"
	InstanceLabelKey  = "app.kubernetes.io/instance"
	PartOfLabelKey    = "app.kubernetes.io/part-of"
	ComponentLabelKey = "app.kubernetes.io/component"
	ManagedByLabelKey = "app.kubernetes.io/managed-by"

	MaxCronJobNameLength = 52 // xref: https://github.com/kubernetes/kubernetes/pull/52733
)

var labelKeyBlacklist = []string{
	NameLabelKey,
	VersionLabelKey,
	InstanceLabelKey,
	// PartOfLabelKey, // propagate part-of key
	// ComponentLabelKey, // propagate part-of key
	ManagedByLabelKey,
}

// AddLabelBlacklistFlag is for explicitly initializing the flags
func AddLabelBlacklistFlag(fs *pflag.FlagSet) {
	if fs == nil {
		fs = pflag.CommandLine
	}
	fs.StringSliceVar(&labelKeyBlacklist, "label-key-blacklist", labelKeyBlacklist, "list of keys that are not propagated from a CRD object to its offshoots")
}

func DeleteInBackground() metav1.DeleteOptions {
	policy := metav1.DeletePropagationBackground
	return metav1.DeleteOptions{PropagationPolicy: &policy}
}

func DeleteInForeground() metav1.DeleteOptions {
	policy := metav1.DeletePropagationForeground
	return metav1.DeleteOptions{PropagationPolicy: &policy}
}

func GetKind(v interface{}) string {
	return reflect.Indirect(reflect.ValueOf(v)).Type().Name()
}

func FilterKeys(domainKey string, out, in map[string]string) map[string]string {
	if len(in) == 0 {
		return out
	}
	if out == nil {
		out = make(map[string]string, len(in))
	}

	blacklist := sets.NewString(labelKeyBlacklist...)

	n := len(domainKey)
	var idx int
	for k, v := range in {
		if blacklist.Has(k) {
			continue
		}

		idx = strings.IndexRune(k, '/')
		switch {
		case idx < n:
			out[k] = v
		case idx == n && k[:idx] != domainKey:
			out[k] = v
		case idx > n && k[idx-n-1:idx] != "."+domainKey:
			out[k] = v
		}
	}
	return out
}

func MergeKeys(out, in map[string]string) map[string]string {
	if len(in) == 0 {
		return out
	}
	if out == nil {
		out = make(map[string]string, len(in))
	}

	for k, v := range in {
		if _, ok := out[k]; !ok {
			out[k] = v
		}
	}
	return out
}

func OverwriteKeys(out map[string]string, ins ...map[string]string) map[string]string {
	if len(ins) == 0 {
		return out
	}
	if out == nil {
		n := 0
		for _, in := range ins {
			n += len(in)
		}
		out = make(map[string]string, n)
	}

	for _, in := range ins {
		for k, v := range in {
			out[k] = v
		}
	}
	return out
}

func NameWithPrefix(prefix, name string, customLength ...int) string {
	return ValidNameWithPrefix(prefix, name, customLength...)
}

func ValidNameWithPrefix(prefix, name string, customLength ...int) string {
	maxLength := validation.DNS1123LabelMaxLength
	if len(customLength) != 0 {
		maxLength = customLength[0]
	}
	out := fmt.Sprintf("%s-%s", prefix, name)
	return strings.Trim(out[:min(maxLength, len(out))], "-")
}

func NameWithSuffix(name, suffix string, customLength ...int) string {
	maxLength := validation.DNS1123LabelMaxLength
	if len(customLength) != 0 {
		maxLength = customLength[0]
	}
	if len(suffix) >= maxLength {
		return strings.Trim(suffix[max(0, len(suffix)-maxLength):], "-")
	}
	out := fmt.Sprintf("%s-%s", name[:min(len(name), maxLength-len(suffix)-1)], suffix)
	return strings.Trim(out, "-")
}

// Deprecated: Use NameWithSuffix in new code
func ValidNameWithSuffix(name, suffix string, customLength ...int) string {
	maxLength := validation.DNS1123LabelMaxLength
	if len(customLength) != 0 {
		maxLength = customLength[0]
	}
	out := fmt.Sprintf("%s-%s", name, suffix)
	return strings.Trim(out[max(0, len(out)-maxLength):], "-")
}

func ValidNameWithPrefixNSuffix(prefix, name, suffix string, customLength ...int) string {
	maxLength := validation.DNS1123LabelMaxLength
	if len(customLength) != 0 {
		maxLength = customLength[0]
	}
	out := fmt.Sprintf("%s-%s-%s", prefix, name, suffix)
	n := len(out)
	if n <= maxLength {
		return strings.Trim(out, "-")
	}
	return strings.Trim(out[:(maxLength+1)/2]+out[(n-maxLength/2):], "-")
}

func ValidCronJobNameWithPrefix(prefix, name string) string {
	return ValidNameWithPrefix(prefix, name, MaxCronJobNameLength)
}

func ValidCronJobNameWithSuffix(name, suffix string) string {
	return ValidNameWithSuffix(name, suffix, MaxCronJobNameLength)
}

func ValidCronJobNameWithPrefixNSuffix(prefix, name, suffix string) string {
	return ValidNameWithPrefixNSuffix(prefix, name, suffix, MaxCronJobNameLength)
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
