package kutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	. "github.com/appscode/go/types"
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/mattbaird/jsonpatch"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	apps "k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func PatchStatefulSet(c clientset.Interface, cur *apps.StatefulSet, transform func(*apps.StatefulSet)) (*apps.StatefulSet, error) {
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
	glog.V(5).Infof("Patching StatefulSet %s@%s with %s.", cur.Name, cur.Namespace, string(pb))
	return c.AppsV1beta1().StatefulSets(cur.Namespace).Patch(cur.Name, types.JSONPatchType, pb)
}

func TryPatchStatefulSet(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apps.StatefulSet)) (*apps.StatefulSet, error) {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := c.AppsV1beta1().StatefulSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return cur, err
		} else if err == nil {
			return PatchStatefulSet(c, cur, transform)
		}
		glog.Errorf("Attempt %d failed to patch StatefulSet %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(retryInterval)
	}
	return nil, fmt.Errorf("Failed to patch StatefulSet %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func UpdateStatefulSet(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apps.StatefulSet)) (*apps.StatefulSet, error) {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := c.AppsV1beta1().StatefulSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return cur, err
		} else if err == nil {
			transform(cur)
			return c.AppsV1beta1().StatefulSets(cur.Namespace).Update(cur)
		}
		glog.Errorf("Attempt %d failed to update StatefulSet %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(retryInterval)
	}
	return nil, fmt.Errorf("Failed to update StatefulSet %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func WaitUntilStatefulSetReady(kubeClient clientset.Interface, meta metav1.ObjectMeta) error {
	return backoff.Retry(func() error {
		if obj, err := kubeClient.AppsV1beta1().StatefulSets(meta.Namespace).Get(meta.Name, metav1.GetOptions{}); err == nil {
			if Int32(obj.Spec.Replicas) == obj.Status.Replicas {
				return nil
			}
		}
		return errors.New("check again")
	}, backoff.NewConstantBackOff(2*time.Second))
}
