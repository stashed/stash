/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"context"

	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
)

func CreateOrPatchPod(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Pod) *core.Pod, opts metav1.PatchOptions) (*core.Pod, kutil.VerbType, error) {
	cur, err := c.CoreV1().Pods(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		klog.V(3).Infof("Creating Pod %s/%s.", meta.Namespace, meta.Name)
		out, err := c.CoreV1().Pods(meta.Namespace).Create(ctx, transform(&core.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: core.SchemeGroupVersion.String(),
			},
			ObjectMeta: meta,
		}), metav1.CreateOptions{
			DryRun:       opts.DryRun,
			FieldManager: opts.FieldManager,
		})
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchPod(ctx, c, cur, transform, opts)
}

func PatchPod(ctx context.Context, c kubernetes.Interface, cur *core.Pod, transform func(*core.Pod) *core.Pod, opts metav1.PatchOptions) (*core.Pod, kutil.VerbType, error) {
	return PatchPodObject(ctx, c, cur, transform(cur.DeepCopy()), opts)
}

func PatchPodObject(ctx context.Context, c kubernetes.Interface, cur, mod *core.Pod, opts metav1.PatchOptions) (*core.Pod, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, core.Pod{})
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	klog.V(3).Infof("Patching Pod %s/%s with %s", cur.Namespace, cur.Name, string(patch))
	out, err := c.CoreV1().Pods(cur.Namespace).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	return out, kutil.VerbPatched, err
}

func TryUpdatePod(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Pod) *core.Pod, opts metav1.UpdateOptions) (result *core.Pod, err error) {
	attempt := 0
	err = wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.RetryTimeout, true, func(ctx context.Context) (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().Pods(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.CoreV1().Pods(cur.Namespace).Update(ctx, transform(cur.DeepCopy()), opts)
			return e2 == nil, nil
		}
		klog.Errorf("Attempt %d failed to update Pod %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})
	if err != nil {
		err = errors.Errorf("failed to update Pod %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

// IsPodReady returns true if a pod is ready considering readiness gates; false otherwise.
func IsPodReady(pod *core.Pod) bool {
	conditions := []core.PodConditionType{
		core.PodReady,
	}
	for _, gate := range pod.Spec.ReadinessGates {
		conditions = append(conditions, gate.ConditionType)
	}

	for _, condition := range conditions {
		if !IsPodConditionTrue(pod.Status.Conditions, condition) {
			return false
		}
	}
	return true
}

// ref: https://github.com/coreos/prometheus-operator/blob/c79166fcff3dae7bb8bc1e6bddc81837c2d97c04/pkg/k8sutil/k8sutil.go#L64
// PodRunningAndReady returns whether a pod is running and each container has
// passed it's ready state.
func PodRunningAndReady(pod core.Pod) (bool, error) {
	switch pod.Status.Phase {
	case core.PodFailed, core.PodSucceeded:
		return false, errors.New("pod completed")
	case core.PodRunning:
		if IsPodReady(&pod) {
			return true, nil
		}
		return false, errors.New("pod ready condition not found")
	}
	return false, nil
}

func RestartPods(ctx context.Context, c kubernetes.Interface, namespace string, selector *metav1.LabelSelector) error {
	r, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return err
	}
	return c.CoreV1().Pods(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: r.String(),
	})
}

func WaitUntilPodRunning(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta) error {
	return wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.ReadinessTimeout, true, func(ctx context.Context) (bool, error) {
		if pod, err := c.CoreV1().Pods(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{}); err == nil {
			runningAndReady, _ := PodRunningAndReady(*pod)
			return runningAndReady, nil
		}
		return false, nil
	})
}

func WaitUntilPodRunningBySelector(ctx context.Context, c kubernetes.Interface, namespace string, selector *metav1.LabelSelector, count int) error {
	r, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return err
	}

	return wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.ReadinessTimeout, true, func(ctx context.Context) (bool, error) {
		podList, err := c.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: r.String(),
		})
		if err != nil {
			return false, nil
		}

		if len(podList.Items) != count {
			return false, nil
		}

		for _, pod := range podList.Items {
			runningAndReady, _ := PodRunningAndReady(pod)
			if !runningAndReady {
				return false, nil
			}
		}
		return true, nil
	})
}

func WaitUntilPodDeletedBySelector(ctx context.Context, c kubernetes.Interface, namespace string, selector *metav1.LabelSelector) error {
	sel, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return err
	}

	return wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.ReadinessTimeout, true, func(ctx context.Context) (bool, error) {
		podList, err := c.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: sel.String(),
		})
		if err != nil {
			return false, nil
		}
		return len(podList.Items) == 0, nil
	})
}

// WaitUntillPodTerminatedByLabel waits until all pods with the label are terminated. Timeout is 5 minutes.
func WaitUntillPodTerminatedByLabel(ctx context.Context, c kubernetes.Interface, namespace string, label string) error {
	return wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.GCTimeout, true, func(ctx context.Context) (bool, error) {
		podList, err := c.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: label})
		if err != nil {
			return false, nil
		}
		return len(podList.Items) == 0, nil
	})
}
