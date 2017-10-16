package v1

import (
	"encoding/json"
	"fmt"

	. "github.com/appscode/go/types"
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

func CreateOrPatchRC(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*apiv1.ReplicationController) *apiv1.ReplicationController) (*apiv1.ReplicationController, error) {
	cur, err := c.CoreV1().ReplicationControllers(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating ReplicationController %s/%s.", meta.Namespace, meta.Name)
		return c.CoreV1().ReplicationControllers(meta.Namespace).Create(transform(&apiv1.ReplicationController{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ReplicationController",
				APIVersion: apiv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
	} else if err != nil {
		return nil, err
	}
	return PatchRC(c, cur, transform)
}

func PatchRC(c kubernetes.Interface, cur *apiv1.ReplicationController, transform func(*apiv1.ReplicationController) *apiv1.ReplicationController) (*apiv1.ReplicationController, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur))
	if err != nil {
		return nil, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, apiv1.ReplicationController{})
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	glog.V(3).Infof("Patching ReplicationController %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	return c.CoreV1().ReplicationControllers(cur.Namespace).Patch(cur.Name, types.StrategicMergePatchType, patch)
}

func TryPatchRC(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*apiv1.ReplicationController) *apiv1.ReplicationController) (result *apiv1.ReplicationController, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().ReplicationControllers(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = PatchRC(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch ReplicationController %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to patch ReplicationController %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func TryUpdateRC(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*apiv1.ReplicationController) *apiv1.ReplicationController) (result *apiv1.ReplicationController, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().ReplicationControllers(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.CoreV1().ReplicationControllers(cur.Namespace).Update(transform(cur))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update ReplicationController %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update ReplicationController %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func WaitUntilRCReady(c kubernetes.Interface, meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		if obj, err := c.CoreV1().ReplicationControllers(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			return Int32(obj.Spec.Replicas) == obj.Status.ReadyReplicas, nil
		}

		return false, nil
	})
}
