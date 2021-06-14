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

package v1

import (
	"context"
	"fmt"

	kutil "kmodules.xyz/client-go"
	ocapps "kmodules.xyz/openshift/apis/apps/v1"
	occ "kmodules.xyz/openshift/client/clientset/versioned"
	v1 "kmodules.xyz/webhook-runtime/apis/workload/v1"

	jsonpatch "github.com/evanphx/json-patch"
	jsoniter "github.com/json-iterator/go"
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
	"k8s.io/klog/v2"
)

var json = jsoniter.ConfigFastest

type WorkloadTransformerFunc func(*v1.Workload) *v1.Workload

// WorkloadsGetter has a method to return a WorkloadInterface.
// A group's client should implement this interface.
type WorkloadsGetter interface {
	Workloads(namespace string) WorkloadInterface
}

// WorkloadInterface has methods to work with Workload resources.
type WorkloadInterface interface {
	Create(context.Context, *v1.Workload, metav1.CreateOptions) (*v1.Workload, error)
	Update(context.Context, *v1.Workload, metav1.UpdateOptions) (*v1.Workload, error)
	Delete(ctx context.Context, obj runtime.Object, opts metav1.DeleteOptions) error
	Get(ctx context.Context, obj runtime.Object, opts metav1.GetOptions) (*v1.Workload, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.WorkloadList, error)
	Patch(ctx context.Context, cur *v1.Workload, transform WorkloadTransformerFunc, opts metav1.PatchOptions) (*v1.Workload, kutil.VerbType, error)
	PatchObject(ctx context.Context, cur, mod *v1.Workload, opts metav1.PatchOptions) (*v1.Workload, kutil.VerbType, error)
	CreateOrPatch(ctx context.Context, obj runtime.Object, transform WorkloadTransformerFunc, opts metav1.PatchOptions) (*v1.Workload, kutil.VerbType, error)
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

func (c *workloads) Create(ctx context.Context, w *v1.Workload, opts metav1.CreateOptions) (*v1.Workload, error) {
	var out runtime.Object
	var err error
	switch w.GroupVersionKind() {
	case core.SchemeGroupVersion.WithKind(v1.KindPod):
		obj := &core.Pod{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.CoreV1().Pods(c.ns).Create(ctx, obj, opts)
		// ReplicationController
	case core.SchemeGroupVersion.WithKind(v1.KindReplicationController):
		obj := &core.ReplicationController{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.CoreV1().ReplicationControllers(c.ns).Create(ctx, obj, opts)
		// Deployment
	case extensions.SchemeGroupVersion.WithKind(v1.KindDeployment):
		obj := &extensions.Deployment{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.ExtensionsV1beta1().Deployments(c.ns).Create(ctx, obj, opts)
	case appsv1beta1.SchemeGroupVersion.WithKind(v1.KindDeployment):
		obj := &appsv1beta1.Deployment{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.AppsV1beta1().Deployments(c.ns).Create(ctx, obj, opts)
	case appsv1beta2.SchemeGroupVersion.WithKind(v1.KindDeployment):
		obj := &appsv1beta2.Deployment{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.AppsV1beta2().Deployments(c.ns).Create(ctx, obj, opts)
	case appsv1.SchemeGroupVersion.WithKind(v1.KindDeployment):
		obj := &appsv1.Deployment{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.AppsV1().Deployments(c.ns).Create(ctx, obj, opts)
		// DaemonSet
	case extensions.SchemeGroupVersion.WithKind(v1.KindDaemonSet):
		obj := &extensions.DaemonSet{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.ExtensionsV1beta1().DaemonSets(c.ns).Create(ctx, obj, opts)
	case appsv1beta2.SchemeGroupVersion.WithKind(v1.KindDaemonSet):
		obj := &appsv1beta2.DaemonSet{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.AppsV1beta2().DaemonSets(c.ns).Create(ctx, obj, opts)
	case appsv1.SchemeGroupVersion.WithKind(v1.KindDaemonSet):
		obj := &appsv1.DaemonSet{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.AppsV1().DaemonSets(c.ns).Create(ctx, obj, opts)
		// ReplicaSet
	case extensions.SchemeGroupVersion.WithKind(v1.KindReplicaSet):
		obj := &extensions.ReplicaSet{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.ExtensionsV1beta1().ReplicaSets(c.ns).Create(ctx, obj, opts)
	case appsv1beta2.SchemeGroupVersion.WithKind(v1.KindReplicaSet):
		obj := &appsv1beta2.ReplicaSet{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.AppsV1beta2().ReplicaSets(c.ns).Create(ctx, obj, opts)
	case appsv1.SchemeGroupVersion.WithKind(v1.KindReplicaSet):
		obj := &appsv1.ReplicaSet{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.AppsV1().ReplicaSets(c.ns).Create(ctx, obj, opts)
		// StatefulSet
	case appsv1beta1.SchemeGroupVersion.WithKind(v1.KindStatefulSet):
		obj := &appsv1beta1.StatefulSet{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.AppsV1beta1().StatefulSets(c.ns).Create(ctx, obj, opts)
	case appsv1beta2.SchemeGroupVersion.WithKind(v1.KindStatefulSet):
		obj := &appsv1beta2.StatefulSet{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.AppsV1beta2().StatefulSets(c.ns).Create(ctx, obj, opts)
	case appsv1.SchemeGroupVersion.WithKind(v1.KindStatefulSet):
		obj := &appsv1.StatefulSet{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.AppsV1().StatefulSets(c.ns).Create(ctx, obj, opts)
		// Job
	case batchv1.SchemeGroupVersion.WithKind(v1.KindJob):
		obj := &batchv1.Job{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.BatchV1().Jobs(c.ns).Create(ctx, obj, opts)
		// CronJob
	case batchv1beta1.SchemeGroupVersion.WithKind(v1.KindCronJob):
		obj := &batchv1beta1.CronJob{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.kc.BatchV1beta1().CronJobs(c.ns).Create(ctx, obj, opts)
	case ocapps.SchemeGroupVersion.WithKind(v1.KindDeploymentConfig):
		obj := &ocapps.DeploymentConfig{}
		if err = ApplyWorkload(obj, w); err != nil {
			return nil, err
		}
		out, err = c.oc.AppsV1().DeploymentConfigs(c.ns).Create(ctx, obj, opts)
	default:
		err = fmt.Errorf("the object is not a pod or does not have a pod template")
	}
	if err != nil {
		return nil, err
	}
	return ConvertToWorkload(out)
}

func (c *workloads) Update(ctx context.Context, w *v1.Workload, opts metav1.UpdateOptions) (*v1.Workload, error) {
	var out runtime.Object
	var err error
	switch t := w.Object.(type) {
	case *core.Pod:
		out, err = c.kc.CoreV1().Pods(c.ns).Update(ctx, t, opts)
		// ReplicationController
	case *core.ReplicationController:
		out, err = c.kc.CoreV1().ReplicationControllers(c.ns).Update(ctx, t, opts)
		// Deployment
	case *extensions.Deployment:
		out, err = c.kc.ExtensionsV1beta1().Deployments(c.ns).Update(ctx, t, opts)
	case *appsv1beta1.Deployment:
		out, err = c.kc.AppsV1beta1().Deployments(c.ns).Update(ctx, t, opts)
	case *appsv1beta2.Deployment:
		out, err = c.kc.AppsV1beta2().Deployments(c.ns).Update(ctx, t, opts)
	case *appsv1.Deployment:
		out, err = c.kc.AppsV1().Deployments(c.ns).Update(ctx, t, opts)
		// DaemonSet
	case *extensions.DaemonSet:
		out, err = c.kc.ExtensionsV1beta1().DaemonSets(c.ns).Update(ctx, t, opts)
	case *appsv1beta2.DaemonSet:
		out, err = c.kc.AppsV1beta2().DaemonSets(c.ns).Update(ctx, t, opts)
	case *appsv1.DaemonSet:
		out, err = c.kc.AppsV1().DaemonSets(c.ns).Update(ctx, t, opts)
		// ReplicaSet
	case *extensions.ReplicaSet:
		out, err = c.kc.ExtensionsV1beta1().ReplicaSets(c.ns).Update(ctx, t, opts)
	case *appsv1beta2.ReplicaSet:
		out, err = c.kc.AppsV1beta2().ReplicaSets(c.ns).Update(ctx, t, opts)
	case *appsv1.ReplicaSet:
		out, err = c.kc.AppsV1().ReplicaSets(c.ns).Update(ctx, t, opts)
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		out, err = c.kc.AppsV1beta1().StatefulSets(c.ns).Update(ctx, t, opts)
	case *appsv1beta2.StatefulSet:
		out, err = c.kc.AppsV1beta2().StatefulSets(c.ns).Update(ctx, t, opts)
	case *appsv1.StatefulSet:
		out, err = c.kc.AppsV1().StatefulSets(c.ns).Update(ctx, t, opts)
		// Job
	case *batchv1.Job:
		out, err = c.kc.BatchV1().Jobs(c.ns).Update(ctx, t, opts)
		// CronJob
	case *batchv1beta1.CronJob:
		out, err = c.kc.BatchV1beta1().CronJobs(c.ns).Update(ctx, t, opts)
	case *ocapps.DeploymentConfig:
		out, err = c.oc.AppsV1().DeploymentConfigs(c.ns).Update(ctx, t, opts)
	default:
		err = fmt.Errorf("the object is not a pod or does not have a pod template")
	}
	if err != nil {
		return nil, err
	}
	return ConvertToWorkload(out)
}

func (c *workloads) Delete(ctx context.Context, obj runtime.Object, options metav1.DeleteOptions) error {
	switch t := obj.(type) {
	case *core.Pod:
		return c.kc.CoreV1().Pods(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
		// ReplicationController
	case *core.ReplicationController:
		return c.kc.CoreV1().ReplicationControllers(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
		// Deployment
	case *extensions.Deployment:
		return c.kc.ExtensionsV1beta1().Deployments(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
	case *appsv1beta1.Deployment:
		return c.kc.AppsV1beta1().Deployments(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
	case *appsv1beta2.Deployment:
		return c.kc.AppsV1beta2().Deployments(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
	case *appsv1.Deployment:
		return c.kc.AppsV1().Deployments(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
		// DaemonSet
	case *extensions.DaemonSet:
		return c.kc.ExtensionsV1beta1().DaemonSets(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
	case *appsv1beta2.DaemonSet:
		return c.kc.AppsV1beta2().DaemonSets(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
	case *appsv1.DaemonSet:
		return c.kc.AppsV1().DaemonSets(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
		// ReplicaSet
	case *extensions.ReplicaSet:
		return c.kc.ExtensionsV1beta1().ReplicaSets(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
	case *appsv1beta2.ReplicaSet:
		return c.kc.AppsV1beta2().ReplicaSets(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
	case *appsv1.ReplicaSet:
		return c.kc.AppsV1().ReplicaSets(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		return c.kc.AppsV1beta1().StatefulSets(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
	case *appsv1beta2.StatefulSet:
		return c.kc.AppsV1beta2().StatefulSets(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
	case *appsv1.StatefulSet:
		return c.kc.AppsV1().StatefulSets(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
		// Job
	case *batchv1.Job:
		return c.kc.BatchV1().Jobs(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
		// CronJob
	case *batchv1beta1.CronJob:
		return c.kc.BatchV1beta1().CronJobs(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
	case *ocapps.DeploymentConfig:
		return c.oc.AppsV1().DeploymentConfigs(c.ns).Delete(ctx, t.ObjectMeta.Name, options)
	default:
		return fmt.Errorf("the object is not a pod or does not have a pod template")
	}
}

func (c *workloads) Get(ctx context.Context, obj runtime.Object, opts metav1.GetOptions) (*v1.Workload, error) {
	var out runtime.Object
	var err error
	switch t := obj.(type) {
	case *core.Pod:
		out, err = c.kc.CoreV1().Pods(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
		// ReplicationController
	case *core.ReplicationController:
		out, err = c.kc.CoreV1().ReplicationControllers(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
		// Deployment
	case *extensions.Deployment:
		out, err = c.kc.ExtensionsV1beta1().Deployments(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
	case *appsv1beta1.Deployment:
		out, err = c.kc.AppsV1beta1().Deployments(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
	case *appsv1beta2.Deployment:
		out, err = c.kc.AppsV1beta2().Deployments(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
	case *appsv1.Deployment:
		out, err = c.kc.AppsV1().Deployments(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
		// DaemonSet
	case *extensions.DaemonSet:
		out, err = c.kc.ExtensionsV1beta1().DaemonSets(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
	case *appsv1beta2.DaemonSet:
		out, err = c.kc.AppsV1beta2().DaemonSets(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
	case *appsv1.DaemonSet:
		out, err = c.kc.AppsV1().DaemonSets(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
		// ReplicaSet
	case *extensions.ReplicaSet:
		out, err = c.kc.ExtensionsV1beta1().ReplicaSets(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
	case *appsv1beta2.ReplicaSet:
		out, err = c.kc.AppsV1beta2().ReplicaSets(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
	case *appsv1.ReplicaSet:
		out, err = c.kc.AppsV1().ReplicaSets(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		out, err = c.kc.AppsV1beta1().StatefulSets(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
	case *appsv1beta2.StatefulSet:
		out, err = c.kc.AppsV1beta2().StatefulSets(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
	case *appsv1.StatefulSet:
		out, err = c.kc.AppsV1().StatefulSets(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
		// Job
	case *batchv1.Job:
		out, err = c.kc.BatchV1().Jobs(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
		// CronJob
	case *batchv1beta1.CronJob:
		out, err = c.kc.BatchV1beta1().CronJobs(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
	case *ocapps.DeploymentConfig:
		out, err = c.oc.AppsV1().DeploymentConfigs(c.ns).Get(ctx, t.ObjectMeta.Name, opts)
	default:
		err = fmt.Errorf("the object is not a pod or does not have a pod template")
	}
	if err != nil {
		return nil, err
	}
	return ConvertToWorkload(out)
}

func (c *workloads) List(ctx context.Context, opts metav1.ListOptions) (*v1.WorkloadList, error) {
	options := metav1.ListOptions{
		LabelSelector:   opts.LabelSelector,
		FieldSelector:   opts.FieldSelector,
		ResourceVersion: opts.ResourceVersion,
		TimeoutSeconds:  opts.TimeoutSeconds,
	}
	list := v1.WorkloadList{Items: make([]v1.Workload, 0)}

	if c.kc != nil {
		{
			objects, err := c.kc.AppsV1beta1().Deployments(c.ns).List(ctx, options)
			if err != nil {
				return nil, err
			}
			err = meta.EachListItem(objects, func(obj runtime.Object) error {
				w, err := ConvertToWorkload(obj)
				if err != nil {
					return err
				}
				list.Items = append(list.Items, *w)
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
		{
			objects, err := c.kc.ExtensionsV1beta1().ReplicaSets(c.ns).List(ctx, options)
			if err != nil {
				return nil, err
			}
			err = meta.EachListItem(objects, func(obj runtime.Object) error {
				w, err := ConvertToWorkload(obj)
				if err != nil {
					return err
				}
				list.Items = append(list.Items, *w)
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
		{
			if c.kc != nil {
				objects, err := c.kc.AppsV1beta1().StatefulSets(c.ns).List(ctx, options)
				if err != nil {
					return nil, err
				}
				err = meta.EachListItem(objects, func(obj runtime.Object) error {
					w, err := ConvertToWorkload(obj)
					if err != nil {
						return err
					}
					list.Items = append(list.Items, *w)
					return nil
				})
				if err != nil {
					return nil, err
				}
			}
		}
		{
			objects, err := c.kc.ExtensionsV1beta1().DaemonSets(c.ns).List(ctx, options)
			if err != nil {
				return nil, err
			}
			err = meta.EachListItem(objects, func(obj runtime.Object) error {
				w, err := ConvertToWorkload(obj)
				if err != nil {
					return err
				}
				list.Items = append(list.Items, *w)
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
		{
			objects, err := c.kc.CoreV1().ReplicationControllers(c.ns).List(ctx, options)
			if err != nil {
				return nil, err
			}
			err = meta.EachListItem(objects, func(obj runtime.Object) error {
				w, err := ConvertToWorkload(obj)
				if err != nil {
					return err
				}
				list.Items = append(list.Items, *w)
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
		{
			objects, err := c.kc.BatchV1().Jobs(c.ns).List(ctx, options)
			if err != nil {
				return nil, err
			}
			err = meta.EachListItem(objects, func(obj runtime.Object) error {
				w, err := ConvertToWorkload(obj)
				if err != nil {
					return err
				}
				list.Items = append(list.Items, *w)
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
		{
			objects, err := c.kc.BatchV1beta1().CronJobs(c.ns).List(ctx, options)
			if err != nil {
				return nil, err
			}
			err = meta.EachListItem(objects, func(obj runtime.Object) error {
				w, err := ConvertToWorkload(obj)
				if err != nil {
					return err
				}
				list.Items = append(list.Items, *w)
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
	}
	{
		if c.oc != nil {
			objects, err := c.oc.AppsV1().DeploymentConfigs(c.ns).List(ctx, options)
			if err != nil {
				return nil, err
			}
			err = meta.EachListItem(objects, func(obj runtime.Object) error {
				w, err := ConvertToWorkload(obj)
				if err != nil {
					return err
				}
				list.Items = append(list.Items, *w)
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
	}

	return &list, nil
}

func (c *workloads) Patch(ctx context.Context, cur *v1.Workload, transform WorkloadTransformerFunc, opts metav1.PatchOptions) (*v1.Workload, kutil.VerbType, error) {
	mod := transform(cur.DeepCopy())
	err := ApplyWorkload(mod.Object, mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return c.PatchObject(ctx, cur, mod, opts)
}

func (c *workloads) PatchObject(ctx context.Context, cur, mod *v1.Workload, opts metav1.PatchOptions) (*v1.Workload, kutil.VerbType, error) {
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
	klog.V(3).Infof("Patching workload %s/%s with %s.", cur.Namespace, cur.Name, string(patch))

	var out runtime.Object
	switch mod.Object.(type) {
	case *core.Pod:
		out, err = c.kc.CoreV1().Pods(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
		// ReplicationController
	case *core.ReplicationController:
		out, err = c.kc.CoreV1().ReplicationControllers(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
		// Deployment
	case *extensions.Deployment:
		out, err = c.kc.ExtensionsV1beta1().Deployments(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	case *appsv1beta1.Deployment:
		out, err = c.kc.AppsV1beta1().Deployments(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	case *appsv1beta2.Deployment:
		out, err = c.kc.AppsV1beta2().Deployments(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	case *appsv1.Deployment:
		out, err = c.kc.AppsV1().Deployments(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
		// DaemonSet
	case *extensions.DaemonSet:
		out, err = c.kc.ExtensionsV1beta1().DaemonSets(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	case *appsv1beta2.DaemonSet:
		out, err = c.kc.AppsV1beta2().DaemonSets(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	case *appsv1.DaemonSet:
		out, err = c.kc.AppsV1().DaemonSets(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
		// ReplicaSet
	case *extensions.ReplicaSet:
		out, err = c.kc.ExtensionsV1beta1().ReplicaSets(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	case *appsv1beta2.ReplicaSet:
		out, err = c.kc.AppsV1beta2().ReplicaSets(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	case *appsv1.ReplicaSet:
		out, err = c.kc.AppsV1().ReplicaSets(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
		// StatefulSet
	case *appsv1beta1.StatefulSet:
		out, err = c.kc.AppsV1beta1().StatefulSets(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	case *appsv1beta2.StatefulSet:
		out, err = c.kc.AppsV1beta2().StatefulSets(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	case *appsv1.StatefulSet:
		out, err = c.kc.AppsV1().StatefulSets(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
		// Job
	case *batchv1.Job:
		out, err = c.kc.BatchV1().Jobs(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
		// CronJob
	case *batchv1beta1.CronJob:
		out, err = c.kc.BatchV1beta1().CronJobs(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	case *ocapps.DeploymentConfig:
		out, err = c.oc.AppsV1().DeploymentConfigs(c.ns).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	default:
		err = fmt.Errorf("the object is not a pod or does not have a pod template")
	}
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	result, err := ConvertToWorkload(out)
	return result, kutil.VerbPatched, err
}

func (c *workloads) CreateOrPatch(ctx context.Context, obj runtime.Object, transform WorkloadTransformerFunc, opts metav1.PatchOptions) (*v1.Workload, kutil.VerbType, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.String() == "" {
		return nil, kutil.VerbUnchanged, fmt.Errorf("obj missing GroupVersionKind")
	}

	cur, err := c.Get(ctx, obj, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		name, err := meta.NewAccessor().Name(obj)
		if err != nil {
			return nil, kutil.VerbUnchanged, err
		}
		klog.V(3).Infof("Creating %s %s/%s.", gvk, c.ns, name)
		out, err := c.Create(ctx, transform(&v1.Workload{
			TypeMeta: metav1.TypeMeta{
				Kind:       gvk.Kind,
				APIVersion: gvk.GroupVersion().String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: c.ns,
				Name:      name,
			},
		}), metav1.CreateOptions{
			DryRun:       opts.DryRun,
			FieldManager: opts.FieldManager,
		})
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return c.Patch(ctx, cur, transform, opts)
}
