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

	v1 "kmodules.xyz/client-go/batch/v1"
	"kmodules.xyz/client-go/batch/v1beta1"
	"kmodules.xyz/client-go/discovery"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kutil "kmodules.xyz/client-go"
)

const kindCronJob = "CronJob"

func CreateOrPatchCronJob(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*batchv1.CronJob) *batchv1.CronJob, opts metav1.PatchOptions) (*batchv1.CronJob, kutil.VerbType, error) {
	if discovery.ExistsGroupVersionKind(c.Discovery(), batchv1.SchemeGroupVersion.String(), kindCronJob) {
		return v1.CreateOrPatchCronJob(ctx, c, meta, transform, opts)
	}

	p, vt, err := v1beta1.CreateOrPatchCronJob(
		ctx,
		c,
		meta,
		func(in *batchv1beta1.CronJob) *batchv1beta1.CronJob {
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

func PatchCronJob(ctx context.Context, c kubernetes.Interface, cur *batchv1.CronJob, transform func(*batchv1.CronJob) *batchv1.CronJob, opts metav1.PatchOptions) (*batchv1.CronJob, kutil.VerbType, error) {
	if discovery.ExistsGroupVersionKind(c.Discovery(), batchv1.SchemeGroupVersion.String(), kindCronJob) {
		return v1.PatchCronJob(ctx, c, cur, transform, opts)
	}

	p, vt, err := v1beta1.PatchCronJob(
		ctx,
		c,
		convert_spec_v1_to_v1beta1(cur),
		func(in *batchv1beta1.CronJob) *batchv1beta1.CronJob {
			return convert_spec_v1_to_v1beta1(transform(convert_spec_v1beta1_to_v1(in)))
		},
		opts,
	)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return convert_v1beta1_to_v1(p), vt, nil
}

func CreateCronJob(ctx context.Context, c kubernetes.Interface, in *batchv1.CronJob) (*batchv1.CronJob, error) {
	if discovery.ExistsGroupVersionKind(c.Discovery(), batchv1.SchemeGroupVersion.String(), kindCronJob) {
		return c.BatchV1().CronJobs(in.Namespace).Create(ctx, in, metav1.CreateOptions{})
	}
	result, err := c.BatchV1beta1().CronJobs(in.Namespace).Create(ctx, convert_spec_v1_to_v1beta1(in), metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return convert_v1beta1_to_v1(result), nil
}

func GetCronJob(ctx context.Context, c kubernetes.Interface, meta types.NamespacedName) (*batchv1.CronJob, error) {
	if discovery.ExistsGroupVersionKind(c.Discovery(), batchv1.SchemeGroupVersion.String(), kindCronJob) {
		return c.BatchV1().CronJobs(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	}
	result, err := c.BatchV1beta1().CronJobs(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return convert_v1beta1_to_v1(result), nil
}

func ListCronJob(ctx context.Context, c kubernetes.Interface, ns string, opts metav1.ListOptions) (*batchv1.CronJobList, error) {
	if discovery.ExistsGroupVersionKind(c.Discovery(), batchv1.SchemeGroupVersion.String(), kindCronJob) {
		return c.BatchV1().CronJobs(ns).List(ctx, opts)
	}
	result, err := c.BatchV1beta1().CronJobs(ns).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	out := batchv1.CronJobList{
		TypeMeta: result.TypeMeta,
		ListMeta: result.ListMeta,
		Items:    make([]batchv1.CronJob, 0, len(result.Items)),
	}
	for _, item := range result.Items {
		out.Items = append(out.Items, *convert_v1beta1_to_v1(&item))
	}
	return &out, nil
}

func DeleteCronJob(ctx context.Context, c kubernetes.Interface, meta types.NamespacedName) error {
	if discovery.ExistsGroupVersionKind(c.Discovery(), batchv1.SchemeGroupVersion.String(), kindCronJob) {
		return c.BatchV1().CronJobs(meta.Namespace).Delete(ctx, meta.Name, metav1.DeleteOptions{})
	}
	return c.BatchV1beta1().CronJobs(meta.Namespace).Delete(ctx, meta.Name, metav1.DeleteOptions{})
}

func convert_spec_v1beta1_to_v1(in *batchv1beta1.CronJob) *batchv1.CronJob {
	return &batchv1.CronJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       in.Kind,
			APIVersion: batchv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: in.ObjectMeta,
		Spec: batchv1.CronJobSpec{
			Schedule:                in.Spec.Schedule,
			TimeZone:                in.Spec.TimeZone,
			StartingDeadlineSeconds: in.Spec.StartingDeadlineSeconds,
			ConcurrencyPolicy:       batchv1.ConcurrencyPolicy(in.Spec.ConcurrencyPolicy),
			Suspend:                 in.Spec.Suspend,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: in.Spec.JobTemplate.ObjectMeta,
				Spec:       in.Spec.JobTemplate.Spec,
			},
			SuccessfulJobsHistoryLimit: in.Spec.SuccessfulJobsHistoryLimit,
			FailedJobsHistoryLimit:     in.Spec.FailedJobsHistoryLimit,
		},
	}
}

func convert_v1beta1_to_v1(in *batchv1beta1.CronJob) *batchv1.CronJob {
	out := convert_spec_v1beta1_to_v1(in)
	out.Status = batchv1.CronJobStatus{
		Active:             in.Status.Active,
		LastScheduleTime:   in.Status.LastScheduleTime,
		LastSuccessfulTime: in.Status.LastSuccessfulTime,
	}
	return out
}

func convert_spec_v1_to_v1beta1(in *batchv1.CronJob) *batchv1beta1.CronJob {
	return &batchv1beta1.CronJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       in.Kind,
			APIVersion: batchv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: in.ObjectMeta,
		Spec: batchv1beta1.CronJobSpec{
			Schedule:                in.Spec.Schedule,
			TimeZone:                in.Spec.TimeZone,
			StartingDeadlineSeconds: in.Spec.StartingDeadlineSeconds,
			ConcurrencyPolicy:       batchv1beta1.ConcurrencyPolicy(in.Spec.ConcurrencyPolicy),
			Suspend:                 in.Spec.Suspend,
			JobTemplate: batchv1beta1.JobTemplateSpec{
				ObjectMeta: in.Spec.JobTemplate.ObjectMeta,
				Spec:       in.Spec.JobTemplate.Spec,
			},
			SuccessfulJobsHistoryLimit: in.Spec.SuccessfulJobsHistoryLimit,
			FailedJobsHistoryLimit:     in.Spec.FailedJobsHistoryLimit,
		},
	}
}

func convert_v1_to_v1beta1(in *batchv1.CronJob) *batchv1beta1.CronJob {
	out := convert_spec_v1_to_v1beta1(in)
	out.Status = batchv1beta1.CronJobStatus{
		Active:             in.Status.Active,
		LastScheduleTime:   in.Status.LastScheduleTime,
		LastSuccessfulTime: in.Status.LastSuccessfulTime,
	}
	return out
}
