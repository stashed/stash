package v1

import (
	"encoding/json"
	"fmt"

	"github.com/appscode/kutil"
	"github.com/golang/glog"
	apiv1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func CreateOrPatchServiceAccount(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*apiv1.ServiceAccount) *apiv1.ServiceAccount) (*apiv1.ServiceAccount, error) {
	cur, err := c.CoreV1().ServiceAccounts(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating ServiceAccount %s/%s.", meta.Namespace, meta.Name)
		return c.CoreV1().ServiceAccounts(meta.Namespace).Create(transform(&apiv1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ServiceAccount",
				APIVersion: apiv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
	} else if err != nil {
		return nil, err
	}
	return PatchServiceAccount(c, cur, transform)
}

func PatchServiceAccount(c kubernetes.Interface, cur *apiv1.ServiceAccount, transform func(*apiv1.ServiceAccount) *apiv1.ServiceAccount) (*apiv1.ServiceAccount, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur))
	if err != nil {
		return nil, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, apiv1.ServiceAccount{})
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	glog.V(3).Infof("Patching ServiceAccount %s/%s with %s", cur.Namespace, cur.Name, string(patch))
	return c.CoreV1().ServiceAccounts(cur.Namespace).Patch(cur.Name, types.StrategicMergePatchType, patch)
}

func TryPatchServiceAccount(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*apiv1.ServiceAccount) *apiv1.ServiceAccount) (result *apiv1.ServiceAccount, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().ServiceAccounts(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = PatchServiceAccount(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch ServiceAccount %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to patch ServiceAccount %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func TryUpdateServiceAccount(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*apiv1.ServiceAccount) *apiv1.ServiceAccount) (result *apiv1.ServiceAccount, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().ServiceAccounts(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.CoreV1().ServiceAccounts(cur.Namespace).Update(transform(cur))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update ServiceAccount %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update ServiceAccount %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}
