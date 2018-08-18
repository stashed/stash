package scale

import (
	"fmt"
	"strconv"

	"github.com/appscode/go/log"
	"github.com/appscode/go/types"
	"github.com/appscode/kutil"
	apps_util "github.com/appscode/kutil/apps/v1"
	core_util "github.com/appscode/kutil/core/v1"
	meta_util "github.com/appscode/kutil/meta"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/backup"
	"github.com/appscode/stash/pkg/util"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type Options struct {
	Workload  api.LocalTypedReference
	Namespace string
	Selector  string
}

type Controller struct {
	k8sClient kubernetes.Interface
	opt       Options
	locked    chan struct{}
}

var (
	ZeroReplica = int32(0)
	OneReplica  = int32(1)
)

func New(k8sClient kubernetes.Interface, opt Options) *Controller {
	return &Controller{
		k8sClient: k8sClient,
		opt:       opt,
	}
}

func (c *Controller) ScaleDownWorkload() error {

	// scale down deployment to 0 replica
	dpList, err := c.k8sClient.AppsV1().Deployments(c.opt.Namespace).List(metav1.ListOptions{LabelSelector: c.opt.Selector})
	if err == nil {
		for _, dp := range dpList.Items {
			_, _, err := apps_util.PatchDeployment(c.k8sClient, &dp, func(obj *apps.Deployment) *apps.Deployment {
				if obj.Annotations == nil {
					obj.Annotations = make(map[string]string)
				}
				obj.Annotations[util.AnnotationOldReplica] = strconv.Itoa(int(*dp.Spec.Replicas))
				obj.Spec.Replicas = &ZeroReplica
				return obj
			})
			if err != nil {
				return err
			}
		}
	}

	// scale down replication controller to 0 replica
	rcList, err := c.k8sClient.CoreV1().ReplicationControllers(c.opt.Namespace).List(metav1.ListOptions{LabelSelector: c.opt.Selector})
	if err == nil {
		for _, rc := range rcList.Items {
			_, _, err := core_util.PatchRC(c.k8sClient, &rc, func(obj *core.ReplicationController) *core.ReplicationController {
				if obj.Annotations == nil {
					obj.Annotations = make(map[string]string)
				}
				obj.Annotations[util.AnnotationOldReplica] = strconv.Itoa(int(*rc.Spec.Replicas))
				obj.Spec.Replicas = &ZeroReplica
				return obj
			})
			if err != nil {
				return err
			}
		}
	}

	// scale down replicaset to 0 replica
	rsList, err := c.k8sClient.AppsV1().ReplicaSets(c.opt.Namespace).List(metav1.ListOptions{LabelSelector: c.opt.Selector})
	if err == nil {
		for _, rs := range rsList.Items {
			if !apps_util.IsOwnedByDeployment(rs.OwnerReferences) {
				_, _, err := apps_util.PatchReplicaSet(c.k8sClient, &rs, func(obj *apps.ReplicaSet) *apps.ReplicaSet {
					if obj.Annotations == nil {
						obj.Annotations = make(map[string]string)
					}
					obj.Annotations[util.AnnotationOldReplica] = strconv.Itoa(int(*rs.Spec.Replicas))
					obj.Spec.Replicas = &ZeroReplica
					return obj
				})
				if err != nil {
					return err
				}
			}
		}
	}

	// wait until workloads are scaled down
	err = c.waitUntilScaledDown()
	if err != nil {
		log.Infof(err.Error())
	}

	//scale up deployment to 1 replica
	dpList, err = c.k8sClient.AppsV1().Deployments(c.opt.Namespace).List(metav1.ListOptions{LabelSelector: c.opt.Selector})
	if err == nil && len(dpList.Items) > 0 {
		for _, dp := range dpList.Items {
			_, _, err := apps_util.PatchDeployment(c.k8sClient, &dp, func(obj *apps.Deployment) *apps.Deployment {
				obj.Spec.Replicas = &OneReplica
				return obj
			})
			if err != nil {
				return err
			}
		}
	}

	//scale up replication controller to 1 replica
	rcList, err = c.k8sClient.CoreV1().ReplicationControllers(c.opt.Namespace).List(metav1.ListOptions{LabelSelector: c.opt.Selector})
	if err == nil && len(rcList.Items) > 0 {
		for _, rc := range rcList.Items {
			_, _, err := core_util.PatchRC(c.k8sClient, &rc, func(obj *core.ReplicationController) *core.ReplicationController {
				obj.Spec.Replicas = &OneReplica
				return obj
			})
			if err != nil {
				return err
			}
		}
	}

	//scale up replicaset to 1 replica
	rsList, err = c.k8sClient.AppsV1().ReplicaSets(c.opt.Namespace).List(metav1.ListOptions{LabelSelector: c.opt.Selector})
	if err == nil && len(rsList.Items) > 0 {
		for _, rs := range rsList.Items {
			if !apps_util.IsOwnedByDeployment(rs.OwnerReferences) {
				_, _, err := apps_util.PatchReplicaSet(c.k8sClient, &rs, func(obj *apps.ReplicaSet) *apps.ReplicaSet {
					obj.Spec.Replicas = &OneReplica
					return obj
				})
				if err != nil {
					return err
				}
			}
		}
	}

	// delete all pods of daemonset and statefulset so that they restart with init container
	podList, err := c.k8sClient.CoreV1().Pods(c.opt.Namespace).List(metav1.ListOptions{LabelSelector: c.opt.Selector})
	if err == nil && len(podList.Items) > 0 {
		for _, pod := range podList.Items {
			if isDaemonOrStatefulSetPod(pod.OwnerReferences) {
				err = c.k8sClient.CoreV1().Pods(c.opt.Namespace).Delete(pod.Name, meta_util.DeleteInBackground())
				if err != nil {
					log.Infof("Error in deleting pod %v. Reason: %v", pod.Name, err.Error())
				}
			}
		}
	}

	return nil
}

func ScaleUpWorkload(k8sClient *kubernetes.Clientset, opt backup.Options) error {
	switch opt.Workload.Kind {
	case api.KindDeployment:
		obj, err := k8sClient.AppsV1().Deployments(opt.Namespace).Get(opt.Workload.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		replica, err := meta_util.GetIntValue(obj.Annotations, util.AnnotationOldReplica)
		if err != nil {
			return err
		}

		_, _, err = apps_util.PatchDeployment(k8sClient, obj, func(dp *apps.Deployment) *apps.Deployment {
			dp.Spec.Replicas = types.Int32P(int32(replica))
			delete(dp.Annotations, util.AnnotationOldReplica)
			return dp
		})
		if err != nil {
			return err
		}
	case api.KindReplicationController:
		obj, err := k8sClient.CoreV1().ReplicationControllers(opt.Namespace).Get(opt.Workload.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		replica, err := meta_util.GetIntValue(obj.Annotations, util.AnnotationOldReplica)
		if err != nil {
			return err
		}

		_, _, err = core_util.PatchRC(k8sClient, obj, func(rc *core.ReplicationController) *core.ReplicationController {
			rc.Spec.Replicas = types.Int32P(int32(replica))
			delete(rc.Annotations, util.AnnotationOldReplica)
			return rc
		})
		if err != nil {
			return err
		}
	case api.KindReplicaSet:
		obj, err := k8sClient.AppsV1().ReplicaSets(opt.Namespace).Get(opt.Workload.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		replica, err := meta_util.GetIntValue(obj.Annotations, util.AnnotationOldReplica)
		if err != nil {
			return err
		}

		_, _, err = apps_util.PatchReplicaSet(k8sClient, obj, func(rs *apps.ReplicaSet) *apps.ReplicaSet {
			rs.Spec.Replicas = types.Int32P(int32(replica))
			delete(rs.Annotations, util.AnnotationOldReplica)
			return rs
		})
		if err != nil {
			return err
		}
	case api.KindStatefulSet:
		// do nothing. we didn't scale down.
	case api.KindDaemonSet:
		// do nothing.
	default:
		return fmt.Errorf("Unknown workload type")

	}

	return nil
}

func (c *Controller) waitUntilScaledDown() error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.GCTimeout, func() (bool, error) {
		podList, err := c.k8sClient.CoreV1().Pods(c.opt.Namespace).List(metav1.ListOptions{LabelSelector: c.opt.Selector})
		if err != nil {
			return false, nil
		}
		for _, pod := range podList.Items {
			if !isDaemonOrStatefulSetPod(pod.OwnerReferences) {
				return false, nil
			}
		}
		return true, nil
	})
}

func isDaemonOrStatefulSetPod(ownerRefs []metav1.OwnerReference) bool {
	for _, ref := range ownerRefs {
		if ref.Kind == api.KindStatefulSet || ref.Kind == api.KindDaemonSet {
			return true
		}
	}
	return false
}
