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

func EnsureServiceAccount(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.ServiceAccount) *apiv1.ServiceAccount) (*apiv1.ServiceAccount, error) {
	return CreateOrPatchServiceAccount(c, meta, transform)
}

func CreateOrPatchServiceAccount(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.ServiceAccount) *apiv1.ServiceAccount) (*apiv1.ServiceAccount, error) {
	cur, err := c.CoreV1().ServiceAccounts(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		return c.CoreV1().ServiceAccounts(meta.Namespace).Create(transform(&apiv1.ServiceAccount{ObjectMeta: meta}))
	} else if err != nil {
		return nil, err
	}
	return PatchServiceAccount(c, cur, transform)
}

func PatchServiceAccount(c clientset.Interface, cur *apiv1.ServiceAccount, transform func(*apiv1.ServiceAccount) *apiv1.ServiceAccount) (*apiv1.ServiceAccount, error) {
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
	glog.V(5).Infof("Patching ServiceAccount %s@%s.", cur.Name, cur.Namespace)
	return c.CoreV1().ServiceAccounts(cur.Namespace).Patch(cur.Name, types.JSONPatchType, pb)
}

func TryPatchServiceAccount(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.ServiceAccount) *apiv1.ServiceAccount) (*apiv1.ServiceAccount, error) {
	attempt := 0
	for ; attempt < kutil.MaxAttempts; attempt = attempt + 1 {
		cur, err := c.CoreV1().ServiceAccounts(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return cur, err
		} else if err == nil {
			return PatchServiceAccount(c, cur, transform)
		}
		glog.Errorf("Attempt %d failed to patch ServiceAccount %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(kutil.RetryInterval)
	}
	return nil, fmt.Errorf("Failed to patch ServiceAccount %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}

func TryUpdateServiceAccount(c clientset.Interface, meta metav1.ObjectMeta, transform func(*apiv1.ServiceAccount) *apiv1.ServiceAccount) (*apiv1.ServiceAccount, error) {
	attempt := 0
	for ; attempt < kutil.MaxAttempts; attempt = attempt + 1 {
		cur, err := c.CoreV1().ServiceAccounts(meta.Namespace).Get(meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(err) {
			return cur, err
		} else if err == nil {
			return c.CoreV1().ServiceAccounts(cur.Namespace).Update(transform(cur))
		}
		glog.Errorf("Attempt %d failed to update ServiceAccount %s@%s due to %s.", attempt, cur.Name, cur.Namespace, err)
		time.Sleep(kutil.RetryInterval)
	}
	return nil, fmt.Errorf("Failed to update ServiceAccount %s@%s after %d attempts.", meta.Name, meta.Namespace, attempt)
}
