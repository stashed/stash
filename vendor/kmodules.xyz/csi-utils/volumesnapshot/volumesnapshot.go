package batch

import (
	"context"
	"time"

	v1 "kmodules.xyz/csi-utils/volumesnapshot/v1"
	"kmodules.xyz/csi-utils/volumesnapshot/v1beta1"

	apiv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	apiv1beta1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1beta1"
	cs "github.com/kubernetes-csi/external-snapshotter/client/v4/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	kutil "kmodules.xyz/client-go"
	"kmodules.xyz/client-go/discovery"
)

const kindVolumeSnapshot = "VolumeSnapshot"

func CreateOrPatchVolumeSnapshot(ctx context.Context, c cs.Interface, meta metav1.ObjectMeta, transform func(*apiv1.VolumeSnapshot) *apiv1.VolumeSnapshot, opts metav1.PatchOptions) (*apiv1.VolumeSnapshot, kutil.VerbType, error) {
	if discovery.ExistsGroupVersionKind(c.Discovery(), apiv1.SchemeGroupVersion.String(), kindVolumeSnapshot) {
		return v1.CreateOrPatchVolumeSnapshot(ctx, c, meta, transform, opts)
	}

	p, vt, err := v1beta1.CreateOrPatchVolumeSnapshot(
		ctx,
		c,
		meta,
		func(in *apiv1beta1.VolumeSnapshot) *apiv1beta1.VolumeSnapshot {
			out := convert_v1_to_v1beta1(transform(convert_v1beta1_to_v1(in)))
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
	if discovery.ExistsGroupVersionKind(c.Discovery(), apiv1.SchemeGroupVersion.String(), kindVolumeSnapshot) {
		return c.SnapshotV1().VolumeSnapshots(in.Namespace).Create(ctx, in, metav1.CreateOptions{})
	}
	result, err := c.SnapshotV1beta1().VolumeSnapshots(in.Namespace).Create(ctx, convert_v1_to_v1beta1(in), metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return convert_v1beta1_to_v1(result), nil
}

func GetVolumeSnapshot(ctx context.Context, c cs.Interface, meta types.NamespacedName) (*apiv1.VolumeSnapshot, error) {
	if discovery.ExistsGroupVersionKind(c.Discovery(), apiv1.SchemeGroupVersion.String(), kindVolumeSnapshot) {
		return c.SnapshotV1().VolumeSnapshots(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	}
	result, err := c.SnapshotV1beta1().VolumeSnapshots(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return convert_v1beta1_to_v1(result), nil
}

func ListVolumeSnapshot(ctx context.Context, c cs.Interface, ns string, opts metav1.ListOptions) (*apiv1.VolumeSnapshotList, error) {
	if discovery.ExistsGroupVersionKind(c.Discovery(), apiv1.SchemeGroupVersion.String(), kindVolumeSnapshot) {
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
	if discovery.ExistsGroupVersionKind(c.Discovery(), apiv1.SchemeGroupVersion.String(), kindVolumeSnapshot) {
		return c.SnapshotV1().VolumeSnapshots(meta.Namespace).Delete(ctx, meta.Name, metav1.DeleteOptions{})
	}
	return c.SnapshotV1beta1().VolumeSnapshots(meta.Namespace).Delete(ctx, meta.Name, metav1.DeleteOptions{})
}

func WaitUntilVolumeSnapshotReady(c cs.Interface, meta types.NamespacedName) error {
	if discovery.ExistsGroupVersionKind(c.Discovery(), apiv1.SchemeGroupVersion.String(), kindVolumeSnapshot) {
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

func convert_v1beta1_to_v1(in *apiv1beta1.VolumeSnapshot) *apiv1.VolumeSnapshot {
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

func convert_v1_to_v1beta1(in *apiv1.VolumeSnapshot) *apiv1beta1.VolumeSnapshot {
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
