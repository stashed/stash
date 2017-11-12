package v1beta1

import (
	"encoding/json"
	"fmt"

	"github.com/appscode/kutil"
	"github.com/golang/glog"
	rbac "k8s.io/api/rbac/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func CreateOrPatchClusterRoleBinding(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*rbac.ClusterRoleBinding) *rbac.ClusterRoleBinding) (*rbac.ClusterRoleBinding, error) {
	cur, err := c.RbacV1beta1().ClusterRoleBindings().Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating ClusterRoleBinding %s.", meta.Name)
		return c.RbacV1beta1().ClusterRoleBindings().Create(transform(&rbac.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterRoleBinding",
				APIVersion: rbac.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
	} else if err != nil {
		return nil, err
	}
	return PatchClusterRoleBinding(c, cur, transform)
}

func PatchClusterRoleBinding(c kubernetes.Interface, cur *rbac.ClusterRoleBinding, transform func(*rbac.ClusterRoleBinding) *rbac.ClusterRoleBinding) (*rbac.ClusterRoleBinding, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur.DeepCopy()))
	if err != nil {
		return nil, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, rbac.ClusterRoleBinding{})
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	glog.V(3).Infof("Patching ClusterRoleBinding %s with %s.", cur.Name, string(patch))
	return c.RbacV1beta1().ClusterRoleBindings().Patch(cur.Name, types.StrategicMergePatchType, patch)
}

func TryPatchClusterRoleBinding(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*rbac.ClusterRoleBinding) *rbac.ClusterRoleBinding) (result *rbac.ClusterRoleBinding, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.RbacV1beta1().ClusterRoleBindings().Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = PatchClusterRoleBinding(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch ClusterRoleBinding %s due to %v.", attempt, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to patch ClusterRoleBinding %s after %d attempts due to %v", meta.Name, attempt, err)
	}
	return
}

func TryUpdateClusterRoleBinding(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*rbac.ClusterRoleBinding) *rbac.ClusterRoleBinding) (result *rbac.ClusterRoleBinding, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.RbacV1beta1().ClusterRoleBindings().Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.RbacV1beta1().ClusterRoleBindings().Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update ClusterRoleBinding %s due to %v.", attempt, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update Role %s after %d attempts due to %v", meta.Name, attempt, err)
	}
	return
}
