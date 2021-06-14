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

func CreateOrPatchServiceAccount(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.ServiceAccount) *core.ServiceAccount, opts metav1.PatchOptions) (*core.ServiceAccount, kutil.VerbType, error) {
	cur, err := c.CoreV1().ServiceAccounts(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		klog.V(3).Infof("Creating ServiceAccount %s/%s.", meta.Namespace, meta.Name)
		out, err := c.CoreV1().ServiceAccounts(meta.Namespace).Create(ctx, transform(&core.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ServiceAccount",
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
	return PatchServiceAccount(ctx, c, cur, transform, opts)
}

func PatchServiceAccount(ctx context.Context, c kubernetes.Interface, cur *core.ServiceAccount, transform func(*core.ServiceAccount) *core.ServiceAccount, opts metav1.PatchOptions) (*core.ServiceAccount, kutil.VerbType, error) {
	return PatchServiceAccountObject(ctx, c, cur, transform(cur.DeepCopy()), opts)
}

func PatchServiceAccountObject(ctx context.Context, c kubernetes.Interface, cur, mod *core.ServiceAccount, opts metav1.PatchOptions) (*core.ServiceAccount, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, core.ServiceAccount{})
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	klog.V(3).Infof("Patching ServiceAccount %s/%s with %s", cur.Namespace, cur.Name, string(patch))
	out, err := c.CoreV1().ServiceAccounts(cur.Namespace).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	return out, kutil.VerbPatched, err
}

func TryUpdateServiceAccount(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.ServiceAccount) *core.ServiceAccount, opts metav1.UpdateOptions) (result *core.ServiceAccount, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().ServiceAccounts(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.CoreV1().ServiceAccounts(cur.Namespace).Update(ctx, transform(cur.DeepCopy()), opts)
			return e2 == nil, nil
		}
		klog.Errorf("Attempt %d failed to update ServiceAccount %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = errors.Errorf("failed to update ServiceAccount %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func WaitUntillServiceAccountDeleted(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.GCTimeout, func() (bool, error) {
		_, err := c.CoreV1().ServiceAccounts(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
		if err != nil && kerr.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
}

// IsServiceAccountToken returns true if the secret is a valid api token for the service account
func IsServiceAccountToken(secret *core.Secret, sa *core.ServiceAccount) bool {
	if secret.Type != core.SecretTypeServiceAccountToken {
		return false
	}

	name := secret.Annotations[core.ServiceAccountNameKey]
	uid := secret.Annotations[core.ServiceAccountUIDKey]
	if name != sa.Name {
		// Name must match
		return false
	}
	if len(uid) > 0 && uid != string(sa.UID) {
		// If UID is specified, it must match
		return false
	}

	return true
}
