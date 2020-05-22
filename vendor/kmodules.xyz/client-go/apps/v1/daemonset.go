/*
Copyright The Kmodules Authors.

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

	"github.com/golang/glog"
	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	kutil "kmodules.xyz/client-go"
)

func CreateOrPatchDaemonSet(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*apps.DaemonSet) *apps.DaemonSet, opts metav1.PatchOptions) (*apps.DaemonSet, kutil.VerbType, error) {
	cur, err := c.AppsV1().DaemonSets(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		glog.V(3).Infof("Creating DaemonSet %s/%s.", meta.Namespace, meta.Name)
		out, err := c.AppsV1().DaemonSets(meta.Namespace).Create(ctx, transform(&apps.DaemonSet{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DaemonSet",
				APIVersion: apps.SchemeGroupVersion.String(),
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
	return PatchDaemonSet(ctx, c, cur, transform, opts)
}

func PatchDaemonSet(ctx context.Context, c kubernetes.Interface, cur *apps.DaemonSet, transform func(*apps.DaemonSet) *apps.DaemonSet, opts metav1.PatchOptions) (*apps.DaemonSet, kutil.VerbType, error) {
	return PatchDaemonSetObject(ctx, c, cur, transform(cur.DeepCopy()), opts)
}

func PatchDaemonSetObject(ctx context.Context, c kubernetes.Interface, cur, mod *apps.DaemonSet, opts metav1.PatchOptions) (*apps.DaemonSet, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, apps.DaemonSet{})
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	glog.V(3).Infof("Patching DaemonSet %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.AppsV1().DaemonSets(cur.Namespace).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	return out, kutil.VerbPatched, err
}

func TryUpdateDaemonSet(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*apps.DaemonSet) *apps.DaemonSet, opts metav1.UpdateOptions) (result *apps.DaemonSet, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.AppsV1().DaemonSets(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.AppsV1().DaemonSets(cur.Namespace).Update(ctx, transform(cur.DeepCopy()), opts)
			return e2 == nil, nil
		}
		glog.Errorf("Attempt %d failed to update DaemonSet %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = errors.Errorf("failed to update DaemonSet %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func WaitUntilDaemonSetReady(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta) error {
	return wait.PollImmediate(kutil.RetryInterval, kutil.ReadinessTimeout, func() (bool, error) {
		// It takes some time to populate .status field of the DaemonSet after it is being created.
		// If this function is called just after creating a DaemonSet, the Get methond returns an obj with .status field is defaulted to their default values.
		// At this time, "obj.Status.DesiredNumberScheduled" and "obj.Status.NumberReady" both are defaulted to 0 and "obj.Status.DesiredNumberScheduled == obj.Status.NumberReady"
		// returns "true" which is not expected behavior. Hence, we have to ensure that "obj.Status.DesiredNumberScheduled" has been populated with actual value.
		// Warning: If the DaemonSet has any affinity that results no schedulable pod, this function will stuck until timeout.
		if obj, err := c.AppsV1().DaemonSets(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{}); err == nil && obj.Status.DesiredNumberScheduled != 0 {
			return obj.Status.DesiredNumberScheduled == obj.Status.NumberReady, nil
		}
		return false, nil
	})
}
