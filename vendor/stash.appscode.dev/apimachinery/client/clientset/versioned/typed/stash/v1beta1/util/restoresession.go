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

package util

import (
	"context"
	"fmt"

	api_v1beta1 "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
	cs "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1beta1"

	jsonpatch "github.com/evanphx/json-patch"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
)

func CreateOrPatchRestoreSession(ctx context.Context, c cs.StashV1beta1Interface, meta metav1.ObjectMeta, transform func(in *api_v1beta1.RestoreSession) *api_v1beta1.RestoreSession, opts metav1.PatchOptions) (*api_v1beta1.RestoreSession, kutil.VerbType, error) {
	cur, err := c.RestoreSessions(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		klog.V(3).Infof("Creating RestoreSession %s/%s.", meta.Namespace, meta.Name)
		out, err := c.RestoreSessions(meta.Namespace).Create(ctx, transform(&api_v1beta1.RestoreSession{
			TypeMeta: metav1.TypeMeta{
				Kind:       api_v1beta1.ResourceKindRestoreSession,
				APIVersion: api_v1beta1.SchemeGroupVersion.String(),
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
	return PatchRestoreSession(ctx, c, cur, transform, opts)
}

func PatchRestoreSession(ctx context.Context, c cs.StashV1beta1Interface, cur *api_v1beta1.RestoreSession, transform func(*api_v1beta1.RestoreSession) *api_v1beta1.RestoreSession, opts metav1.PatchOptions) (*api_v1beta1.RestoreSession, kutil.VerbType, error) {
	return PatchRestoreSessionObject(ctx, c, cur, transform(cur.DeepCopy()), opts)
}

func PatchRestoreSessionObject(ctx context.Context, c cs.StashV1beta1Interface, cur, mod *api_v1beta1.RestoreSession, opts metav1.PatchOptions) (*api_v1beta1.RestoreSession, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := jsonpatch.CreateMergePatch(curJson, modJson)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	klog.V(3).Infof("Patching RestoreSession %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.RestoreSessions(cur.Namespace).Patch(ctx, cur.Name, types.MergePatchType, patch, opts)
	return out, kutil.VerbPatched, err
}

func TryUpdateRestoreSession(ctx context.Context, c cs.StashV1beta1Interface, meta metav1.ObjectMeta, transform func(*api_v1beta1.RestoreSession) *api_v1beta1.RestoreSession, opts metav1.UpdateOptions) (result *api_v1beta1.RestoreSession, err error) {
	attempt := 0
	err = wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.RetryTimeout, true, func(ctx context.Context) (bool, error) {
		attempt++
		cur, e2 := c.RestoreSessions(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.RestoreSessions(cur.Namespace).Update(ctx, transform(cur.DeepCopy()), opts)
			return e2 == nil, nil
		}
		klog.Errorf("Attempt %d failed to update RestoreSession %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})
	if err != nil {
		err = fmt.Errorf("failed to update RestoreSession %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func UpdateRestoreSessionStatus(
	ctx context.Context,
	c cs.StashV1beta1Interface,
	meta metav1.ObjectMeta,
	transform func(*api_v1beta1.RestoreSessionStatus) (types.UID, *api_v1beta1.RestoreSessionStatus),
	opts metav1.UpdateOptions,
) (result *api_v1beta1.RestoreSession, err error) {
	apply := func(x *api_v1beta1.RestoreSession) *api_v1beta1.RestoreSession {
		uid, updatedStatus := transform(x.Status.DeepCopy())
		// Ignore status update when uid does not match
		if uid != "" && uid != x.UID {
			return x
		}
		out := &api_v1beta1.RestoreSession{
			TypeMeta:   x.TypeMeta,
			ObjectMeta: x.ObjectMeta,
			Spec:       x.Spec,
			Status:     *updatedStatus,
		}
		return out
	}

	attempt := 0
	cur, err := c.RestoreSessions(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.RetryTimeout, true, func(ctx context.Context) (bool, error) {
		attempt++
		var e2 error
		result, e2 = c.RestoreSessions(meta.Namespace).UpdateStatus(ctx, apply(cur), opts)
		if kerr.IsConflict(e2) {
			latest, e3 := c.RestoreSessions(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
			switch {
			case e3 == nil:
				cur = latest
				return false, nil
			case kutil.IsRequestRetryable(e3):
				return false, nil
			default:
				return false, e3
			}
		} else if err != nil && !kutil.IsRequestRetryable(e2) {
			return false, e2
		}
		return e2 == nil, nil
	})
	if err != nil {
		err = fmt.Errorf("failed to update status of RestoreSession %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func UpdateRestoreSessionStatusForHost(ctx context.Context, c cs.StashV1beta1Interface, restoreSession metav1.ObjectMeta, hostStats api_v1beta1.HostRestoreStats, opts metav1.UpdateOptions) (*api_v1beta1.RestoreSession, error) {
	return UpdateRestoreSessionStatus(ctx, c, restoreSession, func(in *api_v1beta1.RestoreSessionStatus) (types.UID, *api_v1beta1.RestoreSessionStatus) {
		// if an entry already exist for this host then update it
		for i, v := range in.Stats {
			if v.Hostname == hostStats.Hostname {
				in.Stats[i] = hostStats
				return "", in
			}
		}
		// no entry for this host. so add a new entry.
		in.Stats = append(in.Stats, hostStats)
		return "", in
	}, opts)
}
