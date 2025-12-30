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

package dynamic

import (
	"context"
	"encoding/json"
	"fmt"

	"kmodules.xyz/client-go/meta"

	"github.com/pkg/errors"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
)

func CreateOrPatch(
	ctx context.Context,
	c dynamic.Interface,
	gvr schema.GroupVersionResource,
	meta metav1.ObjectMeta,
	transform func(*unstructured.Unstructured) *unstructured.Unstructured,
	opts metav1.PatchOptions,
) (*unstructured.Unstructured, kutil.VerbType, error) {
	var ri dynamic.ResourceInterface
	if meta.Namespace == "" {
		ri = c.Resource(gvr)
	} else {
		ri = c.Resource(gvr).Namespace(meta.Namespace)
	}

	cur, err := ri.Get(ctx, meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		klog.V(3).Infof("Creating %s %s/%s.", gvr.String(), meta.Namespace, meta.Name)
		u := &unstructured.Unstructured{}
		u.SetName(meta.Name)
		u.SetNamespace(meta.Namespace)
		out, err := ri.Create(ctx, transform(u), metav1.CreateOptions{
			DryRun:       opts.DryRun,
			FieldManager: opts.FieldManager,
		})
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return Patch(ctx, c, gvr, cur, transform, opts)
}

func Patch(
	ctx context.Context,
	c dynamic.Interface,
	gvr schema.GroupVersionResource,
	cur *unstructured.Unstructured,
	transform func(*unstructured.Unstructured) *unstructured.Unstructured,
	opts metav1.PatchOptions,
) (*unstructured.Unstructured, kutil.VerbType, error) {
	return PatchObject(ctx, c, gvr, cur, transform(cur.DeepCopy()), opts)
}

func PatchObject(
	ctx context.Context,
	c dynamic.Interface,
	gvr schema.GroupVersionResource,
	cur, mod *unstructured.Unstructured,
	opts metav1.PatchOptions,
) (*unstructured.Unstructured, kutil.VerbType, error) {
	var ri dynamic.ResourceInterface
	if cur.GetNamespace() == "" {
		ri = c.Resource(gvr)
	} else {
		ri = c.Resource(gvr).Namespace(cur.GetNamespace())
	}

	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := jsonmergepatch.CreateThreeWayJSONMergePatch(curJson, modJson, curJson)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	klog.V(3).Infof("Patching %s %s/%s with %s.", gvr.String(), cur.GetNamespace(), cur.GetName(), string(patch))
	out, err := ri.Patch(ctx, cur.GetName(), types.MergePatchType, patch, opts)
	return out, kutil.VerbPatched, err
}

func TryUpdate(
	ctx context.Context,
	c dynamic.Interface,
	gvr schema.GroupVersionResource,
	meta metav1.ObjectMeta,
	transform func(*unstructured.Unstructured) *unstructured.Unstructured,
	opts metav1.UpdateOptions,
) (result *unstructured.Unstructured, err error) {
	var ri dynamic.ResourceInterface
	if meta.Namespace == "" {
		ri = c.Resource(gvr)
	} else {
		ri = c.Resource(gvr).Namespace(meta.Namespace)
	}

	attempt := 0
	err = wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.RetryTimeout, true, func(ctx context.Context) (bool, error) {
		attempt++
		cur, e2 := ri.Get(ctx, meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = ri.Update(ctx, transform(cur.DeepCopy()), opts)
			return e2 == nil, nil
		}
		klog.Errorf("Attempt %d failed to update %s %s/%s due to %v.", attempt, gvr.String(), cur.GetNamespace(), cur.GetName(), e2)
		return false, nil
	})
	if err != nil {
		err = errors.Errorf("failed to update %s %s/%s after %d attempts due to %v", gvr.String(), meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func UpdateStatus(
	ctx context.Context,
	c dynamic.Interface,
	gvr schema.GroupVersionResource,
	in *unstructured.Unstructured,
	transform func(*unstructured.Unstructured) *unstructured.Unstructured,
	opts metav1.UpdateOptions,
) (result *unstructured.Unstructured, err error) {
	var ri dynamic.ResourceInterface
	if in.GetNamespace() == "" {
		ri = c.Resource(gvr)
	} else {
		ri = c.Resource(gvr).Namespace(in.GetNamespace())
	}

	attempt := 0
	cur := in.DeepCopy()
	err = wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.RetryTimeout, true, func(ctx context.Context) (bool, error) {
		attempt++
		var e2 error
		result, e2 = ri.UpdateStatus(ctx, transform(cur), opts)
		if kerr.IsConflict(e2) {
			latest, e3 := ri.Get(ctx, in.GetName(), metav1.GetOptions{})
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
		err = fmt.Errorf("failed to update status of %s %s/%s after %d attempts due to %v", gvr.String(), in.GetNamespace(), in.GetName(), attempt, err)
	}
	return
}

func ApplyLabels(ctx context.Context,
	c dynamic.Interface,
	gvr schema.GroupVersionResource,
	key types.NamespacedName,
	labels map[string]string,
	opts metav1.PatchOptions,
) (*unstructured.Unstructured, error) {
	if !meta.IsOfficialType(gvr.Group) {
		return nil, fmt.Errorf("only supports official api types, found %+v", gvr)
	}

	patch := map[string]any{
		"metadata": map[string]any{
			"labels": labels,
		},
	}
	data, err := json.Marshal(patch)
	if err != nil {
		return nil, err
	}
	var ri dynamic.ResourceInterface
	if key.Namespace == "" {
		ri = c.Resource(gvr)
	} else {
		ri = c.Resource(gvr).Namespace(key.Namespace)
	}
	return ri.Patch(ctx, key.Name, types.StrategicMergePatchType, data, opts)
}
