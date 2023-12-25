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

func CreateOrPatchBackupSession(ctx context.Context, c cs.StashV1beta1Interface, meta metav1.ObjectMeta, transform func(in *api_v1beta1.BackupSession) *api_v1beta1.BackupSession, opts metav1.PatchOptions) (*api_v1beta1.BackupSession, kutil.VerbType, error) {
	cur, err := c.BackupSessions(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		klog.V(3).Infof("Creating BackupSession %s/%s.", meta.Namespace, meta.Name)
		out, err := c.BackupSessions(meta.Namespace).Create(ctx, transform(&api_v1beta1.BackupSession{
			TypeMeta: metav1.TypeMeta{
				Kind:       api_v1beta1.ResourceKindBackupSession,
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
	return PatchBackupSession(ctx, c, cur, transform, opts)
}

func PatchBackupSession(ctx context.Context, c cs.StashV1beta1Interface, cur *api_v1beta1.BackupSession, transform func(*api_v1beta1.BackupSession) *api_v1beta1.BackupSession, opts metav1.PatchOptions) (*api_v1beta1.BackupSession, kutil.VerbType, error) {
	return PatchBackupSessionObject(ctx, c, cur, transform(cur.DeepCopy()), opts)
}

func PatchBackupSessionObject(ctx context.Context, c cs.StashV1beta1Interface, cur, mod *api_v1beta1.BackupSession, opts metav1.PatchOptions) (*api_v1beta1.BackupSession, kutil.VerbType, error) {
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
	klog.V(3).Infof("Patching BackupSession %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.BackupSessions(cur.Namespace).Patch(ctx, cur.Name, types.MergePatchType, patch, opts)
	return out, kutil.VerbPatched, err
}

func TryUpdateBackupSession(ctx context.Context, c cs.StashV1beta1Interface, meta metav1.ObjectMeta, transform func(*api_v1beta1.BackupSession) *api_v1beta1.BackupSession, opts metav1.UpdateOptions) (result *api_v1beta1.BackupSession, err error) {
	attempt := 0
	err = wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.RetryTimeout, true, func(ctx context.Context) (bool, error) {
		attempt++
		cur, e2 := c.BackupSessions(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.BackupSessions(cur.Namespace).Update(ctx, transform(cur.DeepCopy()), opts)
			return e2 == nil, nil
		}
		klog.Errorf("Attempt %d failed to update BackupSession %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = fmt.Errorf("failed to update BackupSession %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func UpdateBackupSessionStatus(
	ctx context.Context,
	c cs.StashV1beta1Interface,
	meta metav1.ObjectMeta,
	transform func(*api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus),
	opts metav1.UpdateOptions,
) (result *api_v1beta1.BackupSession, err error) {
	apply := func(x *api_v1beta1.BackupSession) *api_v1beta1.BackupSession {
		uid, updatedStatus := transform(x.Status.DeepCopy())
		// Ignore status update when uid does not match
		if uid != "" && uid != x.UID {
			return x
		}
		out := &api_v1beta1.BackupSession{
			TypeMeta:   x.TypeMeta,
			ObjectMeta: x.ObjectMeta,
			Spec:       x.Spec,
			Status:     *updatedStatus,
		}
		return out
	}

	attempt := 0
	cur, err := c.BackupSessions(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.RetryTimeout, true, func(ctx context.Context) (bool, error) {
		attempt++
		var e2 error
		result, e2 = c.BackupSessions(meta.Namespace).UpdateStatus(ctx, apply(cur), opts)
		if kerr.IsConflict(e2) {
			latest, e3 := c.BackupSessions(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
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
		err = fmt.Errorf("failed to update status of BackupSession %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func UpdateBackupSessionStatusForHost(
	ctx context.Context,
	c cs.StashV1beta1Interface,
	targetRef api_v1beta1.TargetRef,
	backupSession metav1.ObjectMeta,
	hostStats api_v1beta1.HostBackupStats,
	opts metav1.UpdateOptions,
) (*api_v1beta1.BackupSession, error) {
	return UpdateBackupSessionStatus(ctx, c, backupSession, func(in *api_v1beta1.BackupSessionStatus) (types.UID, *api_v1beta1.BackupSessionStatus) {
		in.Targets = UpsertHostForTarget(in.Targets, targetRef, hostStats)
		return "", in
	}, opts)
}

func UpsertHostForTarget(targets []api_v1beta1.BackupTargetStatus, targetRef api_v1beta1.TargetRef, stat api_v1beta1.HostBackupStats) []api_v1beta1.BackupTargetStatus {
	for i, target := range targets {
		if target.Ref.Name == targetRef.Name &&
			target.Ref.Kind == targetRef.Kind {
			targets[i].Stats = UpsertHost(target.Stats, stat)
		}
	}
	return targets
}

func UpsertHost(stats []api_v1beta1.HostBackupStats, stat ...api_v1beta1.HostBackupStats) []api_v1beta1.HostBackupStats {
	upsert := func(s api_v1beta1.HostBackupStats) {
		for i, st := range stats {
			if st.Hostname == s.Hostname {
				stats[i] = s
				return
			}
		}
		stats = append(stats, s)
	}
	for _, s := range stat {
		upsert(s)
	}
	return stats
}
