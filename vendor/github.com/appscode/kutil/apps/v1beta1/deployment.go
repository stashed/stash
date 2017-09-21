package v1beta1

import (
	"encoding/json"
	"fmt"

	. "github.com/appscode/go/types"
	"github.com/appscode/kutil"
	"github.com/golang/glog"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func EnsureDeployment(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apps.Deployment) *apps.Deployment) (*apps.Deployment, error) {
	return CreateOrPatchDeployment(c, meta, transform)
}

func CreateOrPatchDeployment(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apps.Deployment) *apps.Deployment) (*apps.Deployment, error) {
	cur, err := c.AppsV1beta1().Deployments(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return c.AppsV1beta1().Deployments(meta.Namespace).Create(transform(&apps.Deployment{ObjectMeta: meta}))
	} else if err != nil {
		return nil, err
	}
	return PatchDeployment(c, cur, transform)
}

func PatchDeployment(c clientset.Interface, cur *apps.Deployment, transform func(*apps.Deployment) *apps.Deployment) (*apps.Deployment, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur))
	if err != nil {
		return nil, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, apps.Deployment{})
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	glog.V(5).Infof("Patching Deployment %s@%s with %s.", cur.Name, cur.Namespace, string(patch))
	return c.AppsV1beta1().Deployments(cur.Namespace).Patch(cur.Name, types.StrategicMergePatchType, patch)
}

func TryPatchDeployment(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apps.Deployment) *apps.Deployment) (result *apps.Deployment, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.AppsV1beta1().Deployments(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = PatchDeployment(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch Deployment %s@%s due to %v.", attempt, cur.Name, cur.Namespace, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to patch Deployment %s@%s after %d attempts due to %v", meta.Name, meta.Namespace, attempt, err)
	}
	return
}

func TryUpdateDeployment(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apps.Deployment) *apps.Deployment) (result *apps.Deployment, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.AppsV1beta1().Deployments(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.AppsV1beta1().Deployments(cur.Namespace).Update(transform(cur))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update Deployment %s@%s due to %v.", attempt, cur.Name, cur.Namespace, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update Deployment %s@%s after %d attempts due to %v", meta.Name, meta.Namespace, attempt, err)
	}
	return
}

func WaitUntilDeploymentReady(c clientset.Interface, meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		if obj, err := c.AppsV1beta1().Deployments(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			return Int32(obj.Spec.Replicas) == obj.Status.ReadyReplicas, nil
		}
		return false, nil
	})
}
