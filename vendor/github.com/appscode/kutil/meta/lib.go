package meta

import (
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

	n := len(domainKey)
	var idx int
	for k, v := range in {
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
