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

func CreateOrPatchClusterRole(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*rbac.ClusterRole) *rbac.ClusterRole) (*rbac.ClusterRole, error) {
	cur, err := c.RbacV1beta1().ClusterRoles().Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating ClusterRole %s.", meta.Name)
		return c.RbacV1beta1().ClusterRoles().Create(transform(&rbac.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterRole",
				APIVersion: rbac.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
	} else if err != nil {
		return nil, err
	}
	return PatchClusterRole(c, cur, transform)
}

func PatchClusterRole(c kubernetes.Interface, cur *rbac.ClusterRole, transform func(*rbac.ClusterRole) *rbac.ClusterRole) (*rbac.ClusterRole, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur.DeepCopy()))
	if err != nil {
		return nil, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, rbac.ClusterRole{})
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	glog.V(3).Infof("Patching ClusterRole %s with %s.", cur.Name, string(patch))
	return c.RbacV1beta1().ClusterRoles().Patch(cur.Name, types.StrategicMergePatchType, patch)
}

func TryPatchClusterRole(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*rbac.ClusterRole) *rbac.ClusterRole) (result *rbac.ClusterRole, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.RbacV1beta1().ClusterRoles().Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = PatchClusterRole(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch ClusterRole %s due to %v.", attempt, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to patch ClusterRole %s after %d attempts due to %v", meta.Name, attempt, err)
	}
	return
}

func TryUpdateClusterRole(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*rbac.ClusterRole) *rbac.ClusterRole) (result *rbac.ClusterRole, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.RbacV1beta1().ClusterRoles().Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.RbacV1beta1().ClusterRoles().Update(transform(cur.DeepCopy()))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update ClusterRole %s due to %v.", attempt, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update ClusterRole %s after %d attempts due to %v", meta.Name, attempt, err)
	}
	return
}
