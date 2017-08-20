package kutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/mattbaird/jsonpatch"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func PatchDaemonSet(c clientset.Interface, cur *extensions.DaemonSet, transform func(*extensions.DaemonSet)) (*extensions.DaemonSet, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	transform(cur)
	modJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	patch, err := jsonpatch.CreatePatch(curJson, modJson)
	if err != nil {
		return nil, err
	}
	pb, err := json.MarshalIndent(patch, "", "  ")
	if err != nil {
		return nil, err
	}
	glog.V(5).Infof("Patching DaemonSet %s@%s with %s.", cur.Name, cur.Namespace, string(pb))
	return c.ExtensionsV1beta1().DaemonSets(cur.Namespace).Patch(cur.Name, types.JSONPatchType, pb)
}

func TryPatchDaemonSet(c clientset.Interface, meta metav1.ObjectMeta, transform func(*extensions.DaemonSet)) (*extensions.DaemonSet, error) {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := c.ExtensionsV1beta1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return cur, err
		} else if err == nil {
			return PatchDaemonSet(c, cur, transform)
		}
		glog.Errorf("Attempt %d failed to patch DaemonSet %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(updateRetryInterval)
	}
	return nil, fmt.Errorf("Failed to patch DaemonSet %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func TryUpdateDaemonSet(c clientset.Interface, meta metav1.ObjectMeta, transform func(*extensions.DaemonSet)) (*extensions.DaemonSet, error) {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := c.ExtensionsV1beta1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return cur, err
		} else if err == nil {
			transform(cur)
			return c.ExtensionsV1beta1().DaemonSets(cur.Namespace).Update(cur)
		}
		glog.Errorf("Attempt %d failed to update DaemonSet %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(updateRetryInterval)
	}
	return nil, fmt.Errorf("Failed to update DaemonSet %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func WaitUntilDaemonSetReady(kubeClient clientset.Interface, meta metav1.ObjectMeta) error {
	return backoff.Retry(func() error {
		if obj, err := kubeClient.ExtensionsV1beta1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			if obj.Status.DesiredNumberScheduled == obj.Status.NumberReady {
				return nil
			}
		}
		return errors.New("check again")
	}, backoff.NewConstantBackOff(2*time.Second))
}
