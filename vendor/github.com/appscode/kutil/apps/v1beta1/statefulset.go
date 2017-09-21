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

func EnsureStatefulSet(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apps.StatefulSet) *apps.StatefulSet) (*apps.StatefulSet, error) {
	return CreateOrPatchStatefulSet(c, meta, transform)
}

func CreateOrPatchStatefulSet(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apps.StatefulSet) *apps.StatefulSet) (*apps.StatefulSet, error) {
	cur, err := c.AppsV1beta1().StatefulSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return c.AppsV1beta1().StatefulSets(meta.Namespace).Create(transform(&apps.StatefulSet{ObjectMeta: meta}))
	} else if err != nil {
		return nil, err
	}
	return PatchStatefulSet(c, cur, transform)
}

func PatchStatefulSet(c clientset.Interface, cur *apps.StatefulSet, transform func(*apps.StatefulSet) *apps.StatefulSet) (*apps.StatefulSet, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur))
	if err != nil {
		return nil, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, apps.StatefulSet{})
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	glog.V(5).Infof("Patching StatefulSet %s@%s with %s.", cur.Name, cur.Namespace, string(patch))
	return c.AppsV1beta1().StatefulSets(cur.Namespace).Patch(cur.Name, types.StrategicMergePatchType, patch)
}

func TryPatchStatefulSet(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apps.StatefulSet) *apps.StatefulSet) (result *apps.StatefulSet, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.AppsV1beta1().StatefulSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = PatchStatefulSet(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch StatefulSet %s@%s due to %v.", attempt, cur.Name, cur.Namespace, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to patch StatefulSet %s@%s after %d attempts due to %v", meta.Name, meta.Namespace, attempt, err)
	}
	return
}

func TryUpdateStatefulSet(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apps.StatefulSet) *apps.StatefulSet) (result *apps.StatefulSet, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.AppsV1beta1().StatefulSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.AppsV1beta1().StatefulSets(cur.Namespace).Update(transform(cur))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update StatefulSet %s@%s due to %v.", attempt, cur.Name, cur.Namespace, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update StatefulSet %s@%s after %d attempts due to %v", meta.Name, meta.Namespace, attempt, err)
	}
	return
}

func WaitUntilStatefulSetReady(kubeClient clientset.Interface, meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		if obj, err := kubeClient.AppsV1beta1().StatefulSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			return Int32(obj.Spec.Replicas) == obj.Status.Replicas, nil
		}
		return false, nil
	})
}
