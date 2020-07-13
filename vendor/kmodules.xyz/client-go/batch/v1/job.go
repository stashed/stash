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

	"github.com/appscode/go/types"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	batch "k8s.io/api/batch/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	kutil "kmodules.xyz/client-go"
)

func CreateOrPatchJob(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*batch.Job) *batch.Job, opts metav1.PatchOptions) (*batch.Job, kutil.VerbType, error) {
	cur, err := c.BatchV1().Jobs(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating Job %s/%s.", meta.Namespace, meta.Name)
		out, err := c.BatchV1().Jobs(meta.Namespace).Create(ctx, transform(&batch.Job{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Job",
				APIVersion: batch.SchemeGroupVersion.String(),
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
	return PatchJob(ctx, c, cur, transform, opts)
}

func PatchJob(ctx context.Context, c kubernetes.Interface, cur *batch.Job, transform func(*batch.Job) *batch.Job, opts metav1.PatchOptions) (*batch.Job, kutil.VerbType, error) {
	return PatchJobObject(ctx, c, cur, transform(cur.DeepCopy()), opts)
}

func PatchJobObject(ctx context.Context, c kubernetes.Interface, cur, mod *batch.Job, opts metav1.PatchOptions) (*batch.Job, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, batch.Job{})
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	glog.V(3).Infof("Patching Job %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.BatchV1().Jobs(cur.Namespace).Patch(ctx, cur.Name, ktypes.StrategicMergePatchType, patch, opts)
	return out, kutil.VerbPatched, err
}

func TryUpdateJob(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*batch.Job) *batch.Job, opts metav1.UpdateOptions) (result *batch.Job, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.BatchV1().Jobs(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.BatchV1().Jobs(cur.Namespace).Update(ctx, transform(cur.DeepCopy()), opts)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update Job %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = errors.Errorf("failed to update Job %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func WaitUntilJobCompletion(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta) error {
	return wait.PollInfinite(kutil.RetryInterval, func() (bool, error) {
		job, err := c.BatchV1().Jobs(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
		if err != nil {
			if kerr.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		}

		if job.Status.Succeeded > 0 || job.Status.Failed > types.Int32(job.Spec.BackoffLimit) {
			return true, nil
		}
		return false, nil
	})
}
