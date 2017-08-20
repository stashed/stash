package kutil

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/mattbaird/jsonpatch"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func CreateOrPatchSecret(c clientset.Interface, obj *apiv1.Secret) (*apiv1.Secret, error) {
	cur, err := c.CoreV1().Secrets(obj.Namespace).Get(obj.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return c.CoreV1().Secrets(obj.Namespace).Create(obj)
	} else if err != nil {
		return nil, err
	}
	return PatchSecret(c, cur, func(*apiv1.Secret) *apiv1.Secret { return obj })
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

	patch, err := jsonpatch.CreatePatch(curJson, modJson)
	if err != nil {
		return nil, err
	}
	pb, err := json.MarshalIndent(patch, "", "  ")
	if err != nil {
		return nil, err
	}
	glog.V(5).Infof("Patching Secret %s@%s.", cur.Name, cur.Namespace)
	return c.CoreV1().Secrets(cur.Namespace).Patch(cur.Name, types.JSONPatchType, pb)
}

func TryPatchSecret(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.Secret) *apiv1.Secret) (*apiv1.Secret, error) {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := c.CoreV1().Secrets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return cur, err
		} else if err == nil {
			return PatchSecret(c, cur, transform)
		}
		glog.Errorf("Attempt %d failed to patch Secret %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(retryInterval)
	}
	return nil, fmt.Errorf("Failed to patch Secret %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func UpdateSecret(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.Secret) *apiv1.Secret) (*apiv1.Secret, error) {
	attempt := 0
	for ; attempt < maxAttempts; attempt = attempt + 1 {
		cur, err := c.CoreV1().Secrets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return cur, err
		} else if err == nil {
			return c.CoreV1().Secrets(cur.Namespace).Update(transform(cur))
		}
		glog.Errorf("Attempt %d failed to update Secret %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(retryInterval)
	}
	return nil, fmt.Errorf("Failed to update Secret %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}
