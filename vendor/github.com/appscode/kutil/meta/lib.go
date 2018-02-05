package meta

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DeleteInBackground() *metav1.DeleteOptions {
	policy := metav1.DeletePropagationBackground
	return &metav1.DeleteOptions{PropagationPolicy: &policy}
}

func GetKind(v interface{}) string {
	return reflect.Indirect(reflect.ValueOf(v)).Type().Name()
}
