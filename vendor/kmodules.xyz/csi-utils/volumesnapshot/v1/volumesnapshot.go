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
	"time"

	jsonpatch "github.com/evanphx/json-patch/v5"
	api "github.com/kubernetes-csi/external-snapshotter/client/v7/apis/volumesnapshot/v1"
	cs "github.com/kubernetes-csi/external-snapshotter/client/v7/clientset/versioned"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
)

func CreateOrPatchVolumeSnapshot(ctx context.Context, c cs.Interface, meta metav1.ObjectMeta, transform func(*api.VolumeSnapshot) *api.VolumeSnapshot, opts metav1.PatchOptions) (*api.VolumeSnapshot, kutil.VerbType, error) {
	cur, err := c.SnapshotV1().VolumeSnapshots(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		klog.V(3).Infof("Creating VolumeSnapshot %s/%s.", meta.Namespace, meta.Name)
		out, err := c.SnapshotV1().VolumeSnapshots(meta.Namespace).Create(ctx, transform(&api.VolumeSnapshot{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VolumeSnapshot",
				APIVersion: api.SchemeGroupVersion.String(),
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
	return PatchVolumeSnapshot(ctx, c, cur, transform, opts)
}

func PatchVolumeSnapshot(ctx context.Context, c cs.Interface, cur *api.VolumeSnapshot, transform func(*api.VolumeSnapshot) *api.VolumeSnapshot, opts metav1.PatchOptions) (*api.VolumeSnapshot, kutil.VerbType, error) {
	return PatchVolumeSnapshotObject(ctx, c, cur, transform(cur.DeepCopy()), opts)
}

func PatchVolumeSnapshotObject(ctx context.Context, c cs.Interface, cur, mod *api.VolumeSnapshot, opts metav1.PatchOptions) (*api.VolumeSnapshot, kutil.VerbType, error) {
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
	klog.V(3).Infof("Patching VolumeSnapshot %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.SnapshotV1().VolumeSnapshots(cur.Namespace).Patch(ctx, cur.Name, types.MergePatchType, patch, opts)
	return out, kutil.VerbPatched, err
}

func TryUpdateVolumeSnapshot(ctx context.Context, c cs.Interface, meta metav1.ObjectMeta, transform func(*api.VolumeSnapshot) *api.VolumeSnapshot, opts metav1.UpdateOptions) (result *api.VolumeSnapshot, err error) {
	attempt := 0
	err = wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.RetryTimeout, true, func(ctx context.Context) (bool, error) {
		attempt++
		cur, e2 := c.SnapshotV1().VolumeSnapshots(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.SnapshotV1().VolumeSnapshots(cur.Namespace).Update(ctx, transform(cur.DeepCopy()), opts)
			return e2 == nil, nil
		}
		klog.Errorf("Attempt %d failed to update VolumeSnapshot %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = errors.Errorf("failed to update VolumeSnapshot %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func WaitUntilVolumeSnapshotReady(c cs.Interface, meta types.NamespacedName) error {
	return wait.PollUntilContextTimeout(context.TODO(), kutil.RetryInterval, 2*time.Hour, true, func(ctx context.Context) (bool, error) {
		if obj, err := c.SnapshotV1().VolumeSnapshots(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{}); err == nil {
			return obj.Status != nil && obj.Status.ReadyToUse != nil && *obj.Status.ReadyToUse, nil
		}
		return false, nil
	})
}
