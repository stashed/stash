package v1

import (
	"fmt"

	"github.com/appscode/kubernetes-webhook-util/apis/workload/v1"
	"github.com/appscode/kutil"
	"github.com/evanphx/json-patch"
	"github.com/golang/glog"
	"github.com/json-iterator/go"
	ocapps "github.com/openshift/api/apps/v1"
	occ "github.com/openshift/client-go/apps/clientset/versioned"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

var json = jsoniter.ConfigFastest

// WorkloadsGetter has a method to return a WorkloadInterface.
// A group's client should implement this interface.
type WorkloadsGetter interface {
	Workloads(namespace string) WorkloadInterface
}

// WorkloadInterface has methods to work with Workload resources.
type WorkloadInterface interface {
	Create(*v1.Workload) (*v1.Workload, error)
	Update(*v1.Workload) (*v1.Workload, error)
	Delete(obj runtime.Object, options *metav1.DeleteOptions) error
	Get(obj runtime.Object, options metav1.GetOptions) (*v1.Workload, error)
	Patch(cur *v1.Workload, transform func(*v1.Workload) *v1.Workload) (*v1.Workload, kutil.VerbType, error)
	PatchObject(cur, mod *v1.Workload) (*v1.Workload, kutil.VerbType, error)
}

// workloads implements WorkloadInterface
type workloads struct {
	kc kubernetes.Interface
	oc occ.Interface
	ns string
}

var _ WorkloadInterface = &workloads{}

// newWorkloads returns a Workloads
func newWorkloads(kc kubernetes.Interface, oc occ.Interface, namespace string) *workloads {
	return &workloads{
		kc: kc,
		oc: oc,
		ns: namespace,
	}
}

func (c *workloads) Create(w *v1.Workload) (*v1.Workload, error) {
	var out runtime.Object
	var err error
	switch t := w.Object.(type) {
	case *core.Pod:
		out, err = c.kc.CoreV1().Pods(c.ns).Create(t)
		// ReplicationController
	case *core.ReplicationController:
		out, err = c.kc.CoreV1().ReplicationControllers(c.ns).Create(t)
		// Deployment
	case *extensions.Deployment:
		out, err = c.kc.ExtensionsV1beta1().Deployments(c.ns).Create(t)
	case *appsv1beta1.Deployment:
		out, err = c.kc.AppsV1beta1().Deployments(c.ns).Create(t)
	case *appsv1beta2.Deployment:
		out, err = c.kc.AppsV1beta2().Deployments(c.ns).Create(t)
	case *appsv1.Deployment:
		out, err = c.kc.AppsV1().Deployments(c.ns).Create(t)
		// DaemonSet
	case *extensions.DaemonSet:
		out, err = c.kc.ExtensionsV1beta1().DaemonSets(c.ns).Create(t)
	case *appsv1beta2.DaemonSet:
		out, err = c.kc.AppsV1beta2().DaemonSets(c.ns).Create(t)
	case *appsv1.DaemonSet:
		out, err = c.kc.AppsV1().DaemonSets(c.ns).Create(t)
		// ReplicaSet
	case *extensions.ReplicaSet:
		out, err = c.kc.ExtensionsV1beta1().ReplicaSets(c.ns).Create(t)
	case *appsv1beta2.ReplicaSet:
		out, err = c.kc.AppsV1beta2().ReplicaSets(c.ns).Create(t)
	case *appsv1.ReplicaSet:
		out, err = c.kc.AppsV1().ReplicaSets(c.ns).Create(t)
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		out, err = c.kc.AppsV1beta1().StatefulSets(c.ns).Create(t)
	case *appsv1beta2.StatefulSet:
		out, err = c.kc.AppsV1beta2().StatefulSets(c.ns).Create(t)
	case *appsv1.StatefulSet:
		out, err = c.kc.AppsV1().StatefulSets(c.ns).Create(t)
		// Job
	case *batchv1.Job:
		out, err = c.kc.BatchV1().Jobs(c.ns).Create(t)
		// CronJob
	case *batchv1beta1.CronJob:
		out, err = c.kc.BatchV1beta1().CronJobs(c.ns).Create(t)
	case *ocapps.DeploymentConfig:
		out, err = c.oc.AppsV1().DeploymentConfigs(c.ns).Create(t)
	default:
		err = fmt.Errorf("the object is not a pod or does not have a pod template")
	}
	if err != nil {
		return nil, err
	}
	return ConvertToWorkload(out)
}

func (c *workloads) Update(w *v1.Workload) (*v1.Workload, error) {
	var out runtime.Object
	var err error
	switch t := w.Object.(type) {
	case *core.Pod:
		out, err = c.kc.CoreV1().Pods(c.ns).Update(t)
		// ReplicationController
	case *core.ReplicationController:
		out, err = c.kc.CoreV1().ReplicationControllers(c.ns).Update(t)
		// Deployment
	case *extensions.Deployment:
		out, err = c.kc.ExtensionsV1beta1().Deployments(c.ns).Update(t)
	case *appsv1beta1.Deployment:
		out, err = c.kc.AppsV1beta1().Deployments(c.ns).Update(t)
	case *appsv1beta2.Deployment:
		out, err = c.kc.AppsV1beta2().Deployments(c.ns).Update(t)
	case *appsv1.Deployment:
		out, err = c.kc.AppsV1().Deployments(c.ns).Update(t)
		// DaemonSet
	case *extensions.DaemonSet:
		out, err = c.kc.ExtensionsV1beta1().DaemonSets(c.ns).Update(t)
	case *appsv1beta2.DaemonSet:
		out, err = c.kc.AppsV1beta2().DaemonSets(c.ns).Update(t)
	case *appsv1.DaemonSet:
		out, err = c.kc.AppsV1().DaemonSets(c.ns).Update(t)
		// ReplicaSet
	case *extensions.ReplicaSet:
		out, err = c.kc.ExtensionsV1beta1().ReplicaSets(c.ns).Update(t)
	case *appsv1beta2.ReplicaSet:
		out, err = c.kc.AppsV1beta2().ReplicaSets(c.ns).Update(t)
	case *appsv1.ReplicaSet:
		out, err = c.kc.AppsV1().ReplicaSets(c.ns).Update(t)
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		out, err = c.kc.AppsV1beta1().StatefulSets(c.ns).Update(t)
	case *appsv1beta2.StatefulSet:
		out, err = c.kc.AppsV1beta2().StatefulSets(c.ns).Update(t)
	case *appsv1.StatefulSet:
		out, err = c.kc.AppsV1().StatefulSets(c.ns).Update(t)
		// Job
	case *batchv1.Job:
		out, err = c.kc.BatchV1().Jobs(c.ns).Update(t)
		// CronJob
	case *batchv1beta1.CronJob:
		out, err = c.kc.BatchV1beta1().CronJobs(c.ns).Update(t)
	case *ocapps.DeploymentConfig:
		out, err = c.oc.AppsV1().DeploymentConfigs(c.ns).Update(t)
	default:
		err = fmt.Errorf("the object is not a pod or does not have a pod template")
	}
	if err != nil {
		return nil, err
	}
	return ConvertToWorkload(out)
}

func (c *workloads) Delete(obj runtime.Object, options *metav1.DeleteOptions) error {
	switch t := obj.(type) {
	case *core.Pod:
		return c.kc.CoreV1().Pods(c.ns).Delete(t.ObjectMeta.Name, options)
		// ReplicationController
	case *core.ReplicationController:
		return c.kc.CoreV1().ReplicationControllers(c.ns).Delete(t.ObjectMeta.Name, options)
		// Deployment
	case *extensions.Deployment:
		return c.kc.ExtensionsV1beta1().Deployments(c.ns).Delete(t.ObjectMeta.Name, options)
	case *appsv1beta1.Deployment:
		return c.kc.AppsV1beta1().Deployments(c.ns).Delete(t.ObjectMeta.Name, options)
	case *appsv1beta2.Deployment:
		return c.kc.AppsV1beta2().Deployments(c.ns).Delete(t.ObjectMeta.Name, options)
	case *appsv1.Deployment:
		return c.kc.AppsV1().Deployments(c.ns).Delete(t.ObjectMeta.Name, options)
		// DaemonSet
	case *extensions.DaemonSet:
		return c.kc.ExtensionsV1beta1().DaemonSets(c.ns).Delete(t.ObjectMeta.Name, options)
	case *appsv1beta2.DaemonSet:
		return c.kc.AppsV1beta2().DaemonSets(c.ns).Delete(t.ObjectMeta.Name, options)
	case *appsv1.DaemonSet:
		return c.kc.AppsV1().DaemonSets(c.ns).Delete(t.ObjectMeta.Name, options)
		// ReplicaSet
	case *extensions.ReplicaSet:
		return c.kc.ExtensionsV1beta1().ReplicaSets(c.ns).Delete(t.ObjectMeta.Name, options)
	case *appsv1beta2.ReplicaSet:
		return c.kc.AppsV1beta2().ReplicaSets(c.ns).Delete(t.ObjectMeta.Name, options)
	case *appsv1.ReplicaSet:
		return c.kc.AppsV1().ReplicaSets(c.ns).Delete(t.ObjectMeta.Name, options)
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		return c.kc.AppsV1beta1().StatefulSets(c.ns).Delete(t.ObjectMeta.Name, options)
	case *appsv1beta2.StatefulSet:
		return c.kc.AppsV1beta2().StatefulSets(c.ns).Delete(t.ObjectMeta.Name, options)
	case *appsv1.StatefulSet:
		return c.kc.AppsV1().StatefulSets(c.ns).Delete(t.ObjectMeta.Name, options)
		// Job
	case *batchv1.Job:
		return c.kc.BatchV1().Jobs(c.ns).Delete(t.ObjectMeta.Name, options)
		// CronJob
	case *batchv1beta1.CronJob:
		return c.kc.BatchV1beta1().CronJobs(c.ns).Delete(t.ObjectMeta.Name, options)
	case *ocapps.DeploymentConfig:
		return c.oc.AppsV1().DeploymentConfigs(c.ns).Delete(t.ObjectMeta.Name, options)
	default:
		return fmt.Errorf("the object is not a pod or does not have a pod template")
	}
}

func (c *workloads) Get(obj runtime.Object, options metav1.GetOptions) (*v1.Workload, error) {
	var out runtime.Object
	var err error
	switch t := obj.(type) {
	case *core.Pod:
		out, err = c.kc.CoreV1().Pods(c.ns).Get(t.ObjectMeta.Name, options)
		// ReplicationController
	case *core.ReplicationController:
		out, err = c.kc.CoreV1().ReplicationControllers(c.ns).Get(t.ObjectMeta.Name, options)
		// Deployment
	case *extensions.Deployment:
		out, err = c.kc.ExtensionsV1beta1().Deployments(c.ns).Get(t.ObjectMeta.Name, options)
	case *appsv1beta1.Deployment:
		out, err = c.kc.AppsV1beta1().Deployments(c.ns).Get(t.ObjectMeta.Name, options)
	case *appsv1beta2.Deployment:
		out, err = c.kc.AppsV1beta2().Deployments(c.ns).Get(t.ObjectMeta.Name, options)
	case *appsv1.Deployment:
		out, err = c.kc.AppsV1().Deployments(c.ns).Get(t.ObjectMeta.Name, options)
		// DaemonSet
	case *extensions.DaemonSet:
		out, err = c.kc.ExtensionsV1beta1().DaemonSets(c.ns).Get(t.ObjectMeta.Name, options)
	case *appsv1beta2.DaemonSet:
		out, err = c.kc.AppsV1beta2().DaemonSets(c.ns).Get(t.ObjectMeta.Name, options)
	case *appsv1.DaemonSet:
		out, err = c.kc.AppsV1().DaemonSets(c.ns).Get(t.ObjectMeta.Name, options)
		// ReplicaSet
	case *extensions.ReplicaSet:
		out, err = c.kc.ExtensionsV1beta1().ReplicaSets(c.ns).Get(t.ObjectMeta.Name, options)
	case *appsv1beta2.ReplicaSet:
		out, err = c.kc.AppsV1beta2().ReplicaSets(c.ns).Get(t.ObjectMeta.Name, options)
	case *appsv1.ReplicaSet:
		out, err = c.kc.AppsV1().ReplicaSets(c.ns).Get(t.ObjectMeta.Name, options)
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		out, err = c.kc.AppsV1beta1().StatefulSets(c.ns).Get(t.ObjectMeta.Name, options)
	case *appsv1beta2.StatefulSet:
		out, err = c.kc.AppsV1beta2().StatefulSets(c.ns).Get(t.ObjectMeta.Name, options)
	case *appsv1.StatefulSet:
		out, err = c.kc.AppsV1().StatefulSets(c.ns).Get(t.ObjectMeta.Name, options)
		// Job
	case *batchv1.Job:
		out, err = c.kc.BatchV1().Jobs(c.ns).Get(t.ObjectMeta.Name, options)
		// CronJob
	case *batchv1beta1.CronJob:
		out, err = c.kc.BatchV1beta1().CronJobs(c.ns).Get(t.ObjectMeta.Name, options)
	case *ocapps.DeploymentConfig:
		out, err = c.oc.AppsV1().DeploymentConfigs(c.ns).Get(t.ObjectMeta.Name, options)
	default:
		err = fmt.Errorf("the object is not a pod or does not have a pod template")
	}
	if err != nil {
		return nil, err
	}
	return ConvertToWorkload(out)
}

func (c *workloads) Patch(cur *v1.Workload, transform func(*v1.Workload) *v1.Workload) (*v1.Workload, kutil.VerbType, error) {
	return c.PatchObject(cur, transform(cur.DeepCopy()))
}

func (c *workloads) PatchObject(cur, mod *v1.Workload) (*v1.Workload, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur.Object)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod.Object)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := jsonpatch.CreateMergePatch(curJson, modJson)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	glog.V(3).Infof("Patching workload %s/%s with %s.", cur.Namespace, cur.Name, string(patch))

	var out runtime.Object
	switch mod.Object.(type) {
	case *core.Pod:
		out, err = c.kc.CoreV1().Pods(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
		// ReplicationController
	case *core.ReplicationController:
		out, err = c.kc.CoreV1().ReplicationControllers(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
		// Deployment
	case *extensions.Deployment:
		out, err = c.kc.ExtensionsV1beta1().Deployments(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
	case *appsv1beta1.Deployment:
		out, err = c.kc.AppsV1beta1().Deployments(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
	case *appsv1beta2.Deployment:
		out, err = c.kc.AppsV1beta2().Deployments(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
	case *appsv1.Deployment:
		out, err = c.kc.AppsV1().Deployments(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
		// DaemonSet
	case *extensions.DaemonSet:
		out, err = c.kc.ExtensionsV1beta1().DaemonSets(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
	case *appsv1beta2.DaemonSet:
		out, err = c.kc.AppsV1beta2().DaemonSets(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
	case *appsv1.DaemonSet:
		out, err = c.kc.AppsV1().DaemonSets(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
		// ReplicaSet
	case *extensions.ReplicaSet:
		out, err = c.kc.ExtensionsV1beta1().ReplicaSets(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
	case *appsv1beta2.ReplicaSet:
		out, err = c.kc.AppsV1beta2().ReplicaSets(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
	case *appsv1.ReplicaSet:
		out, err = c.kc.AppsV1().ReplicaSets(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		out, err = c.kc.AppsV1beta1().StatefulSets(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
	case *appsv1beta2.StatefulSet:
		out, err = c.kc.AppsV1beta2().StatefulSets(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
	case *appsv1.StatefulSet:
		out, err = c.kc.AppsV1().StatefulSets(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
		// Job
	case *batchv1.Job:
		out, err = c.kc.BatchV1().Jobs(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
		// CronJob
	case *batchv1beta1.CronJob:
		out, err = c.kc.BatchV1beta1().CronJobs(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
	case *ocapps.DeploymentConfig:
		out, err = c.oc.AppsV1().DeploymentConfigs(c.ns).Patch(cur.Name, types.StrategicMergePatchType, patch)
	default:
		err = fmt.Errorf("the object is not a pod or does not have a pod template")
	}
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	result, err := ConvertToWorkload(out)
	return result, kutil.VerbPatched, err
}

func (c *workloads) CreateOrPatchWorkload(obj runtime.Object, transform func(*v1.Workload) *v1.Workload) (*v1.Workload, kutil.VerbType, error) {
	cur, err := c.Get(obj, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		name, err := meta.NewAccessor().Name(obj)
		if err != nil {
			return nil, kutil.VerbUnchanged, err
		}
		gvk := obj.GetObjectKind().GroupVersionKind()
		glog.V(3).Infof("Creating %s %s/%s.", gvk, c.ns, name)
		out, err := c.Create(transform(&v1.Workload{
			TypeMeta: metav1.TypeMeta{
				Kind:       gvk.Kind,
				APIVersion: gvk.GroupVersion().String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: c.ns,
				Name:      name,
			},
		}))
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return c.Patch(cur, transform)
}
