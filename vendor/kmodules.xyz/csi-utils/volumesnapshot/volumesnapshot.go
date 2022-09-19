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

//nolint:unused
package batch

import (
	"context"
	"fmt"
	"time"

	v1 "kmodules.xyz/csi-utils/volumesnapshot/v1"
	"kmodules.xyz/csi-utils/volumesnapshot/v1beta1"

	apiv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	apiv1beta1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1beta1"
	cs "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned"
	"gomodules.xyz/sync"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	kutil "kmodules.xyz/client-go"
	du "kmodules.xyz/client-go/discovery"
)

const kindVolumeSnapshot = "VolumeSnapshot"

var (
	once  sync.Once
	useV1 bool
)

func detectVersion(c discovery.DiscoveryInterface) {
	once.Do(func() error {
		versions, err := du.ListAPIVersions(c, apiv1.SchemeGroupVersion.String(), kindVolumeSnapshot)
		if err != nil {
			return err
		} else if len(versions) == 0 {
			return fmt.Errorf("missing Group=%s Kind=%s", apiv1.GroupName, kindVolumeSnapshot)
		}
		useV1 = sets.NewString(versions...).Has("v1")
		return nil
	})
}

func CreateOrPatchVolumeSnapshot(ctx context.Context, c cs.Interface, meta metav1.ObjectMeta, transform func(*apiv1.VolumeSnapshot) *apiv1.VolumeSnapshot, opts metav1.PatchOptions) (*apiv1.VolumeSnapshot, kutil.VerbType, error) {
	detectVersion(c.Discovery())
	if useV1 {
		return v1.CreateOrPatchVolumeSnapshot(ctx, c, meta, transform, opts)
	}

	p, vt, err := v1beta1.CreateOrPatchVolumeSnapshot(
		ctx,
		c,
		meta,
		func(in *apiv1beta1.VolumeSnapshot) *apiv1beta1.VolumeSnapshot {
			out := convert_spec_v1_to_v1beta1(transform(convert_spec_v1beta1_to_v1(in)))
			out.Status = in.Status
			return out
		},
		opts,
	)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return convert_v1beta1_to_v1(p), vt, nil
}

func CreateVolumeSnapshot(ctx context.Context, c cs.Interface, in *apiv1.VolumeSnapshot) (*apiv1.VolumeSnapshot, error) {
	detectVersion(c.Discovery())
	if useV1 {
		return c.SnapshotV1().VolumeSnapshots(in.Namespace).Create(ctx, in, metav1.CreateOptions{})
	}
	result, err := c.SnapshotV1beta1().VolumeSnapshots(in.Namespace).Create(ctx, convert_spec_v1_to_v1beta1(in), metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return convert_v1beta1_to_v1(result), nil
}

func GetVolumeSnapshot(ctx context.Context, c cs.Interface, meta types.NamespacedName) (*apiv1.VolumeSnapshot, error) {
	detectVersion(c.Discovery())
	if useV1 {
		return c.SnapshotV1().VolumeSnapshots(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	}
	result, err := c.SnapshotV1beta1().VolumeSnapshots(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return convert_v1beta1_to_v1(result), nil
}

func ListVolumeSnapshot(ctx context.Context, c cs.Interface, ns string, opts metav1.ListOptions) (*apiv1.VolumeSnapshotList, error) {
	detectVersion(c.Discovery())
	if useV1 {
		return c.SnapshotV1().VolumeSnapshots(ns).List(ctx, opts)
	}
	result, err := c.SnapshotV1beta1().VolumeSnapshots(ns).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	out := apiv1.VolumeSnapshotList{
		TypeMeta: result.TypeMeta,
		ListMeta: result.ListMeta,
		Items:    make([]apiv1.VolumeSnapshot, 0, len(result.Items)),
	}
	for _, item := range result.Items {
		out.Items = append(out.Items, *convert_v1beta1_to_v1(&item))
	}
	return &out, nil
}

func DeleteVolumeSnapshot(ctx context.Context, c cs.Interface, meta types.NamespacedName) error {
	detectVersion(c.Discovery())
	if useV1 {
		return c.SnapshotV1().VolumeSnapshots(meta.Namespace).Delete(ctx, meta.Name, metav1.DeleteOptions{})
	}
	return c.SnapshotV1beta1().VolumeSnapshots(meta.Namespace).Delete(ctx, meta.Name, metav1.DeleteOptions{})
}

func WaitUntilVolumeSnapshotReady(c cs.Interface, meta types.NamespacedName) error {
	detectVersion(c.Discovery())
	if useV1 {
		return wait.PollImmediate(kutil.RetryInterval, 2*time.Hour, func() (bool, error) {
			if obj, err := c.SnapshotV1().VolumeSnapshots(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{}); err == nil {
				return obj.Status != nil && obj.Status.ReadyToUse != nil && *obj.Status.ReadyToUse, nil
			}
			return false, nil
		})
	}

	return wait.PollImmediate(kutil.RetryInterval, 2*time.Hour, func() (bool, error) {
		if obj, err := c.SnapshotV1beta1().VolumeSnapshots(meta.Namespace).Get(context.TODO(), meta.Name, metav1.GetOptions{}); err == nil {
			return obj.Status != nil && obj.Status.ReadyToUse != nil && *obj.Status.ReadyToUse, nil
		}
		return false, nil
	})
}

func convert_spec_v1beta1_to_v1(in *apiv1beta1.VolumeSnapshot) *apiv1.VolumeSnapshot {
	return &apiv1.VolumeSnapshot{
		TypeMeta: metav1.TypeMeta{
			Kind:       in.Kind,
			APIVersion: apiv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: in.ObjectMeta,
		Spec: apiv1.VolumeSnapshotSpec{
			Source: apiv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: in.Spec.Source.PersistentVolumeClaimName,
				VolumeSnapshotContentName: in.Spec.Source.VolumeSnapshotContentName,
			},
			VolumeSnapshotClassName: in.Spec.VolumeSnapshotClassName,
		},
	}
}

func convert_v1beta1_to_v1(in *apiv1beta1.VolumeSnapshot) *apiv1.VolumeSnapshot {
	out := convert_spec_v1beta1_to_v1(in)
	if in.Status != nil {
		out.Status = &apiv1.VolumeSnapshotStatus{
			BoundVolumeSnapshotContentName: in.Status.BoundVolumeSnapshotContentName,
			CreationTime:                   in.Status.CreationTime,
			ReadyToUse:                     in.Status.ReadyToUse,
			RestoreSize:                    in.Status.RestoreSize,
		}
		if in.Status.Error != nil {
			out.Status.Error = &apiv1.VolumeSnapshotError{
				Time:    in.Status.Error.Time,
				Message: in.Status.Error.Message,
			}
		}
	}
	return out
}

func convert_spec_v1_to_v1beta1(in *apiv1.VolumeSnapshot) *apiv1beta1.VolumeSnapshot {
	return &apiv1beta1.VolumeSnapshot{
		TypeMeta: metav1.TypeMeta{
			Kind:       in.Kind,
			APIVersion: apiv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: in.ObjectMeta,
		Spec: apiv1beta1.VolumeSnapshotSpec{
			Source: apiv1beta1.VolumeSnapshotSource{
				PersistentVolumeClaimName: in.Spec.Source.PersistentVolumeClaimName,
				VolumeSnapshotContentName: in.Spec.Source.VolumeSnapshotContentName,
			},
			VolumeSnapshotClassName: in.Spec.VolumeSnapshotClassName,
		},
	}
}

func convert_v1_to_v1beta1(in *apiv1.VolumeSnapshot) *apiv1beta1.VolumeSnapshot {
	out := convert_spec_v1_to_v1beta1(in)
	if in.Status != nil {
		out.Status = &apiv1beta1.VolumeSnapshotStatus{
			BoundVolumeSnapshotContentName: in.Status.BoundVolumeSnapshotContentName,
			CreationTime:                   in.Status.CreationTime,
			ReadyToUse:                     in.Status.ReadyToUse,
			RestoreSize:                    in.Status.RestoreSize,
		}
		if in.Status.Error != nil {
			out.Status.Error = &apiv1beta1.VolumeSnapshotError{
				Time:    in.Status.Error.Time,
				Message: in.Status.Error.Message,
			}
		}
	}
	return out
}
