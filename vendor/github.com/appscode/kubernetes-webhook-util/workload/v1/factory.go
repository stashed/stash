package v1

import (
	"fmt"

	ocapps "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

func New(t metav1.TypeMeta, o metav1.ObjectMeta, tpl core.PodTemplateSpec) *Workload {
	return &Workload{
		TypeMeta:   t,
		ObjectMeta: o,
		Spec: WorkloadSpec{
			Template: tpl,
		},
	}
}

func newWithObject(t metav1.TypeMeta, o metav1.ObjectMeta, sel *metav1.LabelSelector, tpl core.PodTemplateSpec, obj runtime.Object) *Workload {
	return &Workload{
		TypeMeta:   t,
		ObjectMeta: o,
		Spec: WorkloadSpec{
			Selector: sel,
			Template: tpl,
		},
		Object: obj,
	}
}

// ref: https://github.com/kubernetes/kubernetes/blob/4f083dee54539b0ca24ddc55d53921f5c2efc0b9/pkg/kubectl/cmd/util/factory_client_access.go#L221
func ConvertToWorkload(obj runtime.Object) (*Workload, error) {
	switch t := obj.(type) {
	case *core.Pod:
		return newWithObject(t.TypeMeta, t.ObjectMeta, nil, core.PodTemplateSpec{ObjectMeta: t.ObjectMeta, Spec: t.Spec}, obj), nil
		// ReplicationController
	case *core.ReplicationController:
		if t.Spec.Template == nil {
			t.Spec.Template = &core.PodTemplateSpec{}
		}
		return newWithObject(t.TypeMeta, t.ObjectMeta, &metav1.LabelSelector{MatchLabels: t.Spec.Selector}, *t.Spec.Template, obj), nil
		// Deployment
	case *extensions.Deployment:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1beta1.Deployment:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1beta2.Deployment:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1.Deployment:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Selector, t.Spec.Template, obj), nil
		// DaemonSet
	case *extensions.DaemonSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1beta2.DaemonSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1.DaemonSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Selector, t.Spec.Template, obj), nil
		// ReplicaSet
	case *extensions.ReplicaSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1beta2.ReplicaSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1.ReplicaSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Selector, t.Spec.Template, obj), nil
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1beta2.StatefulSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Selector, t.Spec.Template, obj), nil
	case *appsv1.StatefulSet:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Selector, t.Spec.Template, obj), nil
		// Job
	case *batchv1.Job:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.Selector, t.Spec.Template, obj), nil
		// CronJob
	case *batchv1beta1.CronJob:
		return newWithObject(t.TypeMeta, t.ObjectMeta, t.Spec.JobTemplate.Spec.Selector, t.Spec.JobTemplate.Spec.Template, obj), nil
		// DeploymentConfig
	case *ocapps.DeploymentConfig:
		if t.Spec.Template == nil {
			t.Spec.Template = &core.PodTemplateSpec{}
		}
		return newWithObject(t.TypeMeta, t.ObjectMeta, &metav1.LabelSelector{MatchLabels: t.Spec.Selector}, *t.Spec.Template, obj), nil
	default:
		return nil, fmt.Errorf("the object is not a pod or does not have a pod template")
	}
}

func ApplyWorkload(obj runtime.Object, w *Workload) error {
	switch t := obj.(type) {
	case *core.Pod:
		t.ObjectMeta = w.ObjectMeta
		t.Spec = w.Spec.Template.Spec
		// ReplicationController
	case *core.ReplicationController:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = &w.Spec.Template
		// Deployment
	case *extensions.Deployment:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
	case *appsv1beta1.Deployment:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
	case *appsv1beta2.Deployment:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
	case *appsv1.Deployment:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		// DaemonSet
	case *extensions.DaemonSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
	case *appsv1beta2.DaemonSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
	case *appsv1.DaemonSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		// ReplicaSet
	case *extensions.ReplicaSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
	case *appsv1beta2.ReplicaSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
	case *appsv1.ReplicaSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
	case *appsv1beta2.StatefulSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
	case *appsv1.StatefulSet:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		// Job
	case *batchv1.Job:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = w.Spec.Template
		// CronJob
	case *batchv1beta1.CronJob:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.JobTemplate.Spec.Template = w.Spec.Template
		// DeploymentConfig
	case *ocapps.DeploymentConfig:
		t.ObjectMeta = w.ObjectMeta
		t.Spec.Template = &w.Spec.Template
	default:
		return fmt.Errorf("the object is not a pod or does not have a pod template")
	}
	return nil
}

func Create(client kubernetes.Interface, obj runtime.Object) error {
	var err error
	switch t := obj.(type) {
	case *core.Pod:
		_, err = client.CoreV1().Pods(t.Namespace).Create(t)
		// ReplicationController
	case *core.ReplicationController:
		_, err = client.CoreV1().ReplicationControllers(t.Namespace).Create(t)
		// Deployment
	case *extensions.Deployment:
		_, err = client.ExtensionsV1beta1().Deployments(t.Namespace).Create(t)
	case *appsv1beta1.Deployment:
		_, err = client.AppsV1beta1().Deployments(t.Namespace).Create(t)
	case *appsv1beta2.Deployment:
		_, err = client.AppsV1beta2().Deployments(t.Namespace).Create(t)
	case *appsv1.Deployment:
		_, err = client.AppsV1().Deployments(t.Namespace).Create(t)
		// DaemonSet
	case *extensions.DaemonSet:
		_, err = client.ExtensionsV1beta1().DaemonSets(t.Namespace).Create(t)
	case *appsv1beta2.DaemonSet:
		_, err = client.AppsV1beta2().DaemonSets(t.Namespace).Create(t)
	case *appsv1.DaemonSet:
		_, err = client.AppsV1().DaemonSets(t.Namespace).Create(t)
		// ReplicaSet
	case *extensions.ReplicaSet:
		_, err = client.ExtensionsV1beta1().ReplicaSets(t.Namespace).Create(t)
	case *appsv1beta2.ReplicaSet:
		_, err = client.AppsV1beta2().ReplicaSets(t.Namespace).Create(t)
	case *appsv1.ReplicaSet:
		_, err = client.AppsV1().ReplicaSets(t.Namespace).Create(t)
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		_, err = client.AppsV1beta1().StatefulSets(t.Namespace).Create(t)
	case *appsv1beta2.StatefulSet:
		_, err = client.AppsV1beta2().StatefulSets(t.Namespace).Create(t)
	case *appsv1.StatefulSet:
		_, err = client.AppsV1().StatefulSets(t.Namespace).Create(t)
		// Job
	case *batchv1.Job:
		_, err = client.BatchV1().Jobs(t.Namespace).Create(t)
		// CronJob
	case *batchv1beta1.CronJob:
		_, err = client.BatchV1beta1().CronJobs(t.Namespace).Create(t)
	default:
		err = fmt.Errorf("the object is not a pod or does not have a pod template")
	}
	return err
}

func Update(client kubernetes.Interface, obj runtime.Object) error {
	var err error
	switch t := obj.(type) {
	case *core.Pod:
		_, err = client.CoreV1().Pods(t.Namespace).Update(t)
		// ReplicationController
	case *core.ReplicationController:
		_, err = client.CoreV1().ReplicationControllers(t.Namespace).Update(t)
		// Deployment
	case *extensions.Deployment:
		_, err = client.ExtensionsV1beta1().Deployments(t.Namespace).Update(t)
	case *appsv1beta1.Deployment:
		_, err = client.AppsV1beta1().Deployments(t.Namespace).Update(t)
	case *appsv1beta2.Deployment:
		_, err = client.AppsV1beta2().Deployments(t.Namespace).Update(t)
	case *appsv1.Deployment:
		_, err = client.AppsV1().Deployments(t.Namespace).Update(t)
		// DaemonSet
	case *extensions.DaemonSet:
		_, err = client.ExtensionsV1beta1().DaemonSets(t.Namespace).Update(t)
	case *appsv1beta2.DaemonSet:
		_, err = client.AppsV1beta2().DaemonSets(t.Namespace).Update(t)
	case *appsv1.DaemonSet:
		_, err = client.AppsV1().DaemonSets(t.Namespace).Update(t)
		// ReplicaSet
	case *extensions.ReplicaSet:
		_, err = client.ExtensionsV1beta1().ReplicaSets(t.Namespace).Update(t)
	case *appsv1beta2.ReplicaSet:
		_, err = client.AppsV1beta2().ReplicaSets(t.Namespace).Update(t)
	case *appsv1.ReplicaSet:
		_, err = client.AppsV1().ReplicaSets(t.Namespace).Update(t)
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		_, err = client.AppsV1beta1().StatefulSets(t.Namespace).Update(t)
	case *appsv1beta2.StatefulSet:
		_, err = client.AppsV1beta2().StatefulSets(t.Namespace).Update(t)
	case *appsv1.StatefulSet:
		_, err = client.AppsV1().StatefulSets(t.Namespace).Update(t)
		// Job
	case *batchv1.Job:
		_, err = client.BatchV1().Jobs(t.Namespace).Update(t)
		// CronJob
	case *batchv1beta1.CronJob:
		_, err = client.BatchV1beta1().CronJobs(t.Namespace).Update(t)
	default:
		err = fmt.Errorf("the object is not a pod or does not have a pod template")
	}
	return err
}

func Get(client kubernetes.Interface, obj runtime.Object) error {
	var err error
	switch t := obj.(type) {
	case *core.Pod:
		_, err = client.CoreV1().Pods(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
		// ReplicationController
	case *core.ReplicationController:
		_, err = client.CoreV1().ReplicationControllers(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
		// Deployment
	case *extensions.Deployment:
		_, err = client.ExtensionsV1beta1().Deployments(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
	case *appsv1beta1.Deployment:
		_, err = client.AppsV1beta1().Deployments(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
	case *appsv1beta2.Deployment:
		_, err = client.AppsV1beta2().Deployments(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
	case *appsv1.Deployment:
		_, err = client.AppsV1().Deployments(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
		// DaemonSet
	case *extensions.DaemonSet:
		_, err = client.ExtensionsV1beta1().DaemonSets(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
	case *appsv1beta2.DaemonSet:
		_, err = client.AppsV1beta2().DaemonSets(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
	case *appsv1.DaemonSet:
		_, err = client.AppsV1().DaemonSets(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
		// ReplicaSet
	case *extensions.ReplicaSet:
		_, err = client.ExtensionsV1beta1().ReplicaSets(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
	case *appsv1beta2.ReplicaSet:
		_, err = client.AppsV1beta2().ReplicaSets(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
	case *appsv1.ReplicaSet:
		_, err = client.AppsV1().ReplicaSets(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		_, err = client.AppsV1beta1().StatefulSets(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
	case *appsv1beta2.StatefulSet:
		_, err = client.AppsV1beta2().StatefulSets(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
	case *appsv1.StatefulSet:
		_, err = client.AppsV1().StatefulSets(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
		// Job
	case *batchv1.Job:
		_, err = client.BatchV1().Jobs(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
		// CronJob
	case *batchv1beta1.CronJob:
		_, err = client.BatchV1beta1().CronJobs(t.Namespace).Get(t.ObjectMeta.Name, metav1.GetOptions{})
	default:
		err = fmt.Errorf("the object is not a pod or does not have a pod template")
	}
	return err
}
