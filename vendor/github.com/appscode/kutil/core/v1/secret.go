package v1

import (
	"encoding/json"
	"fmt"

	"github.com/appscode/kutil"
	"github.com/golang/glog"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func EnsureSecret(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.Secret) *apiv1.Secret) (*apiv1.Secret, error) {
	return CreateOrPatchSecret(c, meta, transform)
}

func CreateOrPatchSecret(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.Secret) *apiv1.Secret) (*apiv1.Secret, error) {
	cur, err := c.CoreV1().Secrets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return c.CoreV1().Secrets(meta.Namespace).Create(transform(&apiv1.Secret{ObjectMeta: meta}))
	} else if err != nil {
		return nil, err
	}
	return PatchSecret(c, cur, transform)
}

func PatchSecret(c clientset.Interface, cur *apiv1.Secret, transform func(*apiv1.Secret) *apiv1.Secret) (*apiv1.Secret, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur))
	if err != nil {
		return nil, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, apiv1.Secret{})
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	glog.V(5).Infof("Patching Secret %s@%s.", cur.Name, cur.Namespace)
	return c.CoreV1().Secrets(cur.Namespace).Patch(cur.Name, types.StrategicMergePatchType, patch)
}

func TryPatchSecret(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.Secret) *apiv1.Secret) (result *apiv1.Secret, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().Secrets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = PatchSecret(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch Secret %s@%s due to %v.", attempt, cur.Name, cur.Namespace, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to patch Secret %s@%s after %d attempts due to %v", meta.Name, meta.Namespace, attempt, err)
	}
	return
}

func TryUpdateSecret(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.Secret) *apiv1.Secret) (result *apiv1.Secret, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().Secrets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.CoreV1().Secrets(cur.Namespace).Update(transform(cur))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update Secret %s@%s due to %v.", attempt, cur.Name, cur.Namespace, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update Secret %s@%s after %d attempts due to %v", meta.Name, meta.Namespace, attempt, err)
	}
	return
}
