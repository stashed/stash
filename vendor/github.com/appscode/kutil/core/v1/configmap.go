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

func CreateOrPatchConfigMap(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*apiv1.ConfigMap) *apiv1.ConfigMap) (*apiv1.ConfigMap, error) {
	cur, err := c.CoreV1().ConfigMaps(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating ConfigMap %s/%s.", meta.Namespace, meta.Name)
		return c.CoreV1().ConfigMaps(meta.Namespace).Create(transform(&apiv1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: apiv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
	} else if err != nil {
		return nil, err
	}
	return PatchConfigMap(c, cur, transform)
}

func PatchConfigMap(c kubernetes.Interface, cur *apiv1.ConfigMap, transform func(*apiv1.ConfigMap) *apiv1.ConfigMap) (*apiv1.ConfigMap, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur))
	if err != nil {
		return nil, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, apiv1.ConfigMap{})
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	glog.V(3).Infof("Patching ConfigMap %s/%s with %s", cur.Namespace, cur.Name, string(patch))
	return c.CoreV1().ConfigMaps(cur.Namespace).Patch(cur.Name, types.StrategicMergePatchType, patch)
}

func TryPatchConfigMap(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*apiv1.ConfigMap) *apiv1.ConfigMap) (result *apiv1.ConfigMap, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().ConfigMaps(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = PatchConfigMap(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch ConfigMap %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to patch ConfigMap %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func TryUpdateConfigMap(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*apiv1.ConfigMap) *apiv1.ConfigMap) (result *apiv1.ConfigMap, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().ConfigMaps(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.CoreV1().ConfigMaps(cur.Namespace).Update(transform(cur))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update ConfigMap %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update ConfigMap %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}
