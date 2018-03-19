package v1

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func New(t metav1.TypeMeta, o metav1.ObjectMeta, spec core.PodSpec) *Workload {
	return &Workload{
		TypeMeta:   t,
		ObjectMeta: o,
		Spec:       spec,
	}
}
