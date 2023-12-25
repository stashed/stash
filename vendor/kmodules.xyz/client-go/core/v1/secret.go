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

func CreateOrPatchSecret(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Secret) *core.Secret, opts metav1.PatchOptions, forceSyncType ...bool) (*core.Secret, kutil.VerbType, error) {
	syncType := len(forceSyncType) == 1 && forceSyncType[0]

	cur, err := c.CoreV1().Secrets(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		klog.V(3).Infof("Creating Secret %s/%s.", meta.Namespace, meta.Name)
		out, err := c.CoreV1().Secrets(meta.Namespace).Create(ctx, transform(&core.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
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

	mod := transform(cur.DeepCopy())
	if mod.Type != cur.Type && syncType && len(opts.DryRun) == 0 {
		// secret type can't be modified once created, so we have to delete first, then recreate with correct type
		klog.Warningf("Secret %s/%s type is modified, deleting first.", meta.Namespace, meta.Name)
		foregroundDeletion := metav1.DeletePropagationForeground
		err = c.CoreV1().Secrets(meta.Namespace).Delete(ctx, meta.Name, metav1.DeleteOptions{
			TypeMeta:          metav1.TypeMeta{},
			PropagationPolicy: &foregroundDeletion,
			DryRun:            opts.DryRun,
		})
		if err != nil {
			return nil, kutil.VerbUnchanged, err
		}

		klog.V(3).Infof("Creating Secret %s/%s.", meta.Namespace, meta.Name)
		out, err := c.CoreV1().Secrets(meta.Namespace).Create(ctx, mod, metav1.CreateOptions{
			DryRun:       opts.DryRun,
			FieldManager: opts.FieldManager,
		})
		return out, kutil.VerbCreated, err
	}
	return PatchSecretObject(ctx, c, cur, mod, opts)
}

func PatchSecret(ctx context.Context, c kubernetes.Interface, cur *core.Secret, transform func(*core.Secret) *core.Secret, opts metav1.PatchOptions) (*core.Secret, kutil.VerbType, error) {
	return PatchSecretObject(ctx, c, cur, transform(cur.DeepCopy()), opts)
}

func PatchSecretObject(ctx context.Context, c kubernetes.Interface, cur, mod *core.Secret, opts metav1.PatchOptions) (*core.Secret, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, core.Secret{})
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	klog.V(3).Infof("Patching Secret %s/%s", cur.Namespace, cur.Name)
	out, err := c.CoreV1().Secrets(cur.Namespace).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	return out, kutil.VerbPatched, err
}

func TryUpdateSecret(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Secret) *core.Secret, opts metav1.UpdateOptions) (result *core.Secret, err error) {
	attempt := 0
	err = wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.RetryTimeout, true, func(ctx context.Context) (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().Secrets(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.CoreV1().Secrets(cur.Namespace).Update(ctx, transform(cur.DeepCopy()), opts)
			return e2 == nil, nil
		}
		klog.Errorf("Attempt %d failed to update Secret %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = errors.Errorf("failed to update Secret %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func ObfuscateSecret(in core.Secret) *core.Secret {
	data := make(map[string][]byte)
	for k := range in.Data {
		data[k] = []byte("-")
	}
	in.Data = data
	return &in
}
