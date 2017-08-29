package v1

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/appscode/jsonpatch"
	"github.com/appscode/kutil"
	"github.com/golang/glog"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

func EnsureSecret(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.Secret) *apiv1.Secret) (*apiv1.Secret, error) {
	return CreateOrPatchSecret(c, meta, transform)
}

func CreateOrPatchSecret(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.Secret) *apiv1.Secret) (*apiv1.Secret, error) {
	cur, err := c.CoreV1().Secrets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return c.CoreV1().Secrets(meta.Namespace).Create(transform(&apiv1.Secret{ObjectMeta: meta}))
	} else if err != nil {
		return nil, err
	}
	return PatchSecret(c, cur, transform)
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
	if len(patch) == 0 {
		return cur, nil
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
	for ; attempt < kutil.MaxAttempts; attempt = attempt + 1 {
		cur, err := c.CoreV1().Secrets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return cur, err
		} else if err == nil {
			return PatchSecret(c, cur, transform)
		}
		glog.Errorf("Attempt %d failed to patch Secret %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(kutil.RetryInterval)
	}
	return nil, fmt.Errorf("Failed to patch Secret %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func TryUpdateSecret(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.Secret) *apiv1.Secret) (*apiv1.Secret, error) {
	attempt := 0
	for ; attempt < kutil.MaxAttempts; attempt = attempt + 1 {
		cur, err := c.CoreV1().Secrets(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return cur, err
		} else if err == nil {
			return c.CoreV1().Secrets(cur.Namespace).Update(transform(cur))
		}
		glog.Errorf("Attempt %d failed to update Secret %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(kutil.RetryInterval)
	}
	return nil, fmt.Errorf("Failed to update Secret %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}
