package v1

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func New(t metav1.TypeMeta, o metav1.ObjectMeta, spec core.PodSpec) *Workload {
	return &Workload{
		TypeMeta:   t,
		ObjectMeta: o,
		Spec:       spec,
	}
}

func newWithObject(t metav1.TypeMeta, o metav1.ObjectMeta, spec core.PodSpec, obj runtime.Object) *Workload {
	return &Workload{
		TypeMeta:   t,
		ObjectMeta: o,
		Spec:       spec,
		Object: obj,
	}
}

// ref: https://github.com/kubernetes/kubernetes/blob/4f083dee54539b0ca24ddc55d53921f5c2efc0b9/pkg/kubectl/cmd/util/factory_client_access.go#L221
func ConvertToWorkload(obj runtime.Object) (*Workload, error) {
	switch t := obj.(type) {
	case *core.Pod:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec, obj), nil
		// ReplicationController
	case *core.ReplicationController:
		if t.Spec.Template == nil {
			t.Spec.Template = &core.PodTemplateSpec{}
		}
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Template.Spec, obj), nil
		// Deployment
	case *extensions.Deployment:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Template.Spec, obj), nil
	case *appsv1beta1.Deployment:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Template.Spec, obj), nil
	case *appsv1beta2.Deployment:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Template.Spec, obj), nil
	case *appsv1.Deployment:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Template.Spec, obj), nil
		// DaemonSet
	case *extensions.DaemonSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Template.Spec, obj), nil
	case *appsv1beta2.DaemonSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Template.Spec, obj), nil
	case *appsv1.DaemonSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Template.Spec, obj), nil
		// ReplicaSet
	case *extensions.ReplicaSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Template.Spec, obj), nil
	case *appsv1beta2.ReplicaSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Template.Spec, obj), nil
	case *appsv1.ReplicaSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Template.Spec, obj), nil
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Template.Spec, obj), nil
	case *appsv1beta2.StatefulSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Template.Spec, obj), nil
	case *appsv1.StatefulSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Template.Spec, obj), nil
		// Job
	case *batchv1.Job:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Template.Spec, obj), nil
		// CronJob
	case *batchv1beta1.CronJob:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.JobTemplate.Spec.Template.Spec, obj), nil
	default:
		return nil, fmt.Errorf("the object is not a pod or does not have a pod template")
	}
}

func ApplyWorkload(obj runtime.Object, w *Workload) error {
	switch t := obj.(type) {
	case *core.Pod:
		t.ObjectMeta = w.ObjectMeta
		t.Spec = w.Spec
		// ReplicationController
	case *core.ReplicationController:
		if t.Spec.Template == nil {
			t.Spec.Template = &core.PodTemplateSpec{}
		}
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template.Spec = w.Spec
		// Deployment
	case *extensions.Deployment:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template.Spec = w.Spec
	case *appsv1beta1.Deployment:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template.Spec = w.Spec
	case *appsv1beta2.Deployment:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template.Spec = w.Spec
	case *appsv1.Deployment:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template.Spec = w.Spec
		// DaemonSet
	case *extensions.DaemonSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template.Spec = w.Spec
	case *appsv1beta2.DaemonSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template.Spec = w.Spec
	case *appsv1.DaemonSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template.Spec = w.Spec
		// ReplicaSet
	case *extensions.ReplicaSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template.Spec = w.Spec
	case *appsv1beta2.ReplicaSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template.Spec = w.Spec
	case *appsv1.ReplicaSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template.Spec = w.Spec
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template.Spec = w.Spec
	case *appsv1beta2.StatefulSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template.Spec = w.Spec
	case *appsv1.StatefulSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template.Spec = w.Spec
		// Job
	case *batchv1.Job:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template.Spec = w.Spec
		// CronJob
	case *batchv1beta1.CronJob:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.JobTemplate.Spec.Template.Spec = w.Spec
	default:
		return fmt.Errorf("the object is not a pod or does not have a pod template")
	}
	return nil
}
