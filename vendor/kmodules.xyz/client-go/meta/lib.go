package meta

import (
	"reflect"
	"strings"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
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

func DeleteInBackground() *metav1.DeleteOptions {
	policy := metav1.DeletePropagationBackground
	return &metav1.DeleteOptions{PropagationPolicy: &policy}
}

func GetKind(v interface{}) string {
	return reflect.Indirect(reflect.ValueOf(v)).Type().Name()
}

func FilterKeys(domainKey string, out, in map[string]string) map[string]string {
	if in == nil {
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
