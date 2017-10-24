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

func CreateOrPatchRoleBinding(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*rbac.RoleBinding) *rbac.RoleBinding) (*rbac.RoleBinding, error) {
	cur, err := c.RbacV1beta1().RoleBindings(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating RoleBinding %s/%s.", meta.Namespace, meta.Name)
		return c.RbacV1beta1().RoleBindings(meta.Namespace).Create(transform(&rbac.RoleBinding{
			TypeMeta: metav1.TypeMeta{
				Kind:       "RoleBinding",
				APIVersion: rbac.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
	} else if err != nil {
		return nil, err
	}
	return PatchRoleBinding(c, cur, transform)
}

func PatchRoleBinding(c kubernetes.Interface, cur *rbac.RoleBinding, transform func(*rbac.RoleBinding) *rbac.RoleBinding) (*rbac.RoleBinding, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur))
	if err != nil {
		return nil, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, rbac.RoleBinding{})
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	glog.V(3).Infof("Patching RoleBinding %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	return c.RbacV1beta1().RoleBindings(cur.Namespace).Patch(cur.Name, types.StrategicMergePatchType, patch)
}

func TryPatchRoleBinding(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*rbac.RoleBinding) *rbac.RoleBinding) (result *rbac.RoleBinding, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.RbacV1beta1().RoleBindings(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = PatchRoleBinding(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch RoleBinding %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to patch RoleBinding %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func TryUpdateRoleBinding(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*rbac.RoleBinding) *rbac.RoleBinding) (result *rbac.RoleBinding, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.RbacV1beta1().RoleBindings(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.RbacV1beta1().RoleBindings(cur.Namespace).Update(transform(cur))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update RoleBinding %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update Role %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}
