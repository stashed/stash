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

func EnsureService(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.Service) *apiv1.Service) (*apiv1.Service, error) {
	return CreateOrPatchService(c, meta, transform)
}

func CreateOrPatchService(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.Service) *apiv1.Service) (*apiv1.Service, error) {
	cur, err := c.CoreV1().Services(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return c.CoreV1().Services(meta.Namespace).Create(transform(&apiv1.Service{ObjectMeta: meta}))
	} else if err != nil {
		return nil, err
	}
	return PatchService(c, cur, transform)
}

func PatchService(c clientset.Interface, cur *apiv1.Service, transform func(*apiv1.Service) *apiv1.Service) (*apiv1.Service, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur))
	if err != nil {
		return nil, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, apiv1.Service{})
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	glog.V(5).Infof("Patching Service %s@%s with %s.", cur.Name, cur.Namespace, string(patch))
	return c.CoreV1().Services(cur.Namespace).Patch(cur.Name, types.StrategicMergePatchType, patch)
}

func TryPatchService(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.Service) *apiv1.Service) (result *apiv1.Service, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().Services(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = PatchService(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch Service %s@%s due to %v.", attempt, cur.Name, cur.Namespace, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("Failed to patch Service %s@%s after %d attempts due to %v", meta.Name, meta.Namespace, attempt, err)
	}
	return
}

func TryUpdateService(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.Service) *apiv1.Service) (result *apiv1.Service, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().Services(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.CoreV1().Services(cur.Namespace).Update(transform(cur))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update Service %s@%s due to %v.", attempt, cur.Name, cur.Namespace, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("Failed to update Service %s@%s after %d attempts due to %v", meta.Name, meta.Namespace, attempt, err)
	}
	return
}
