package v1

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/appscode/kutil"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func CreateOrPatchPod(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Pod) *core.Pod) (*core.Pod, error) {
	cur, err := c.CoreV1().Pods(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating Pod %s/%s.", meta.Namespace, meta.Name)
		return c.CoreV1().Pods(meta.Namespace).Create(transform(&core.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: core.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}))
	} else if err != nil {
		return nil, err
	}
	return PatchPod(c, cur, transform)
}

func PatchPod(c kubernetes.Interface, cur *core.Pod, transform func(*core.Pod) *core.Pod) (*core.Pod, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur))
	if err != nil {
		return nil, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, core.Pod{})
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, nil
	}
	glog.V(3).Infof("Patching Pod %s/%s with %s", cur.Namespace, cur.Name, string(patch))
	return c.CoreV1().Pods(cur.Namespace).Patch(cur.Name, types.StrategicMergePatchType, patch)
}

func TryPatchPod(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Pod) *core.Pod) (result *core.Pod, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().Pods(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = PatchPod(c, cur, transform)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to patch Pod %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to patch Pod %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func TryUpdatePod(c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Pod) *core.Pod) (result *core.Pod, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().Pods(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.CoreV1().Pods(cur.Namespace).Update(transform(cur))
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update Pod %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update Pod %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

// ref: https://github.com/coreos/prometheus-operator/blob/c79166fcff3dae7bb8bc1e6bddc81837c2d97c04/pkg/k8sutil/k8sutil.go#L64
// PodRunningAndReady returns whether a pod is running and each container has
// passed it's ready state.
func PodRunningAndReady(pod core.Pod) (bool, error) {
	switch pod.Status.Phase {
	case core.PodFailed, core.PodSucceeded:
		return false, errors.New("pod completed")
	case core.PodRunning:
		for _, cond := range pod.Status.Conditions {
			if cond.Type != core.PodReady {
				continue
			}
			return cond.Status == core.ConditionTrue, nil
		}
		return false, errors.New("pod ready condition not found")
	}
	return false, nil
}

func RestartPods(kubeClient kubernetes.Interface, namespace string, selector *metav1.LabelSelector) error {
	r, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return err
	}
	return kubeClient.CoreV1().Pods(namespace).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: r.String(),
	})
}
