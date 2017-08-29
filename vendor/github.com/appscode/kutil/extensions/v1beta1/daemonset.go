package v1beta1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/appscode/jsonpatch"
	"github.com/appscode/kutil"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func EnsureDaemonSet(c clientset.Interface, meta metav1.ObjectMeta, transform func(*extensions.DaemonSet) *extensions.DaemonSet) (*extensions.DaemonSet, error) {
	return CreateOrPatchDaemonSet(c, meta, transform)
}

func CreateOrPatchDaemonSet(c clientset.Interface, meta metav1.ObjectMeta, transform func(*extensions.DaemonSet) *extensions.DaemonSet) (*extensions.DaemonSet, error) {
	cur, err := c.ExtensionsV1beta1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return c.ExtensionsV1beta1().DaemonSets(meta.Namespace).Create(transform(&extensions.DaemonSet{ObjectMeta: meta}))
	} else if err != nil {
		return nil, err
	}
	return PatchDaemonSet(c, cur, transform)
}

func PatchDaemonSet(c clientset.Interface, cur *extensions.DaemonSet, transform func(*extensions.DaemonSet) *extensions.DaemonSet) (*extensions.DaemonSet, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, err
	}

	modJson, err := json.Marshal(transform(cur))
	if err != nil {
		return nil, err
	}

	patch, err := jsonpatch.CreatePatch(curJson, modJson)
	if err != nil {
		return nil, err
	}
	if len(patch) == 0 {
		return cur, nil
	}
	pb, err := json.MarshalIndent(patch, "", "  ")
	if err != nil {
		return nil, err
	}
	glog.V(5).Infof("Patching DaemonSet %s@%s with %s.", cur.Name, cur.Namespace, string(pb))
	result, err := c.ExtensionsV1beta1().DaemonSets(cur.Namespace).Patch(cur.Name, types.JSONPatchType, pb)
	if ok, err := kutil.CheckAPIVersion(c, "<= 1.5"); err == nil && ok {
		// https://kubernetes.io/docs/tasks/manage-daemon/update-daemon-set/
		core_util.RestartPods(c, cur.Namespace, cur.Spec.Selector)
	}
	return result, err
}

func TryPatchDaemonSet(c clientset.Interface, meta metav1.ObjectMeta, transform func(*extensions.DaemonSet) *extensions.DaemonSet) (*extensions.DaemonSet, error) {
	attempt := 0
	for ; attempt < kutil.MaxAttempts; attempt = attempt + 1 {
		cur, err := c.ExtensionsV1beta1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return cur, err
		} else if err == nil {
			return PatchDaemonSet(c, cur, transform)
		}
		glog.Errorf("Attempt %d failed to patch DaemonSet %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(kutil.RetryInterval)
	}
	return nil, fmt.Errorf("Failed to patch DaemonSet %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func TryUpdateDaemonSet(c clientset.Interface, meta metav1.ObjectMeta, transform func(*extensions.DaemonSet) *extensions.DaemonSet) (*extensions.DaemonSet, error) {
	attempt := 0
	for ; attempt < kutil.MaxAttempts; attempt = attempt + 1 {
		cur, err := c.ExtensionsV1beta1().DaemonSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return cur, err
		} else if err == nil {
			oJson, err := json.Marshal(cur)
			if err != nil {
				return nil, err
			}
			modified := transform(cur)
			mJson, err := json.Marshal(modified)
			if err != nil {
				return nil, err
			}
			if bytes.Equal(oJson, mJson) {
				return cur, err
			}

			result, err := c.ExtensionsV1beta1().DaemonSets(cur.Namespace).Update(transform(cur))
			if ok, err := kutil.CheckAPIVersion(c, "<= 1.5"); err == nil && ok {
				// https://kubernetes.io/docs/tasks/manage-daemon/update-daemon-set/
				core_util.RestartPods(c, cur.Namespace, cur.Spec.Selector)
			}
			return result, err
		}
		glog.Errorf("Attempt %d failed to update DaemonSet %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(kutil.RetryInterval)
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
