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

package v1beta1

import (
	"context"

	"github.com/pkg/errors"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	reg "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	apireg_cs "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	kutil "kmodules.xyz/client-go"
)

func CreateOrPatchAPIService(ctx context.Context, c apireg_cs.Interface, name string, transform func(*reg.APIService) *reg.APIService, opts metav1.PatchOptions) (*reg.APIService, kutil.VerbType, error) {
	cur, err := c.ApiregistrationV1beta1().APIServices().Get(ctx, name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		klog.V(3).Infof("Creating APIService %s.", name)
		out, err := c.ApiregistrationV1beta1().APIServices().Create(ctx, transform(&reg.APIService{
			TypeMeta: metav1.TypeMeta{
				Kind:       "APIService",
				APIVersion: reg.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}), metav1.CreateOptions{
			DryRun:       opts.DryRun,
			FieldManager: opts.FieldManager,
		})
		return out, kutil.VerbCreated, err
	} else if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	return PatchAPIService(ctx, c, cur, transform, opts)
}

func PatchAPIService(ctx context.Context, c apireg_cs.Interface, cur *reg.APIService, transform func(*reg.APIService) *reg.APIService, opts metav1.PatchOptions) (*reg.APIService, kutil.VerbType, error) {
	return PatchAPIServiceObject(ctx, c, cur, transform(cur.DeepCopy()), opts)
}

func PatchAPIServiceObject(ctx context.Context, c apireg_cs.Interface, cur, mod *reg.APIService, opts metav1.PatchOptions) (*reg.APIService, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, reg.APIService{})
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	klog.V(3).Infof("Patching APIService %s with %s.", cur.Name, string(patch))
	out, err := c.ApiregistrationV1beta1().APIServices().Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	return out, kutil.VerbPatched, err
}

func TryUpdateAPIService(ctx context.Context, c apireg_cs.Interface, name string, transform func(*reg.APIService) *reg.APIService, opts metav1.UpdateOptions) (result *reg.APIService, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.ApiregistrationV1beta1().APIServices().Get(ctx, name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.ApiregistrationV1beta1().APIServices().Update(ctx, transform(cur.DeepCopy()), opts)
			return e2 == nil, nil
		}
		klog.Errorf("Attempt %d failed to update APIService %s due to %v.", attempt, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = errors.Errorf("failed to update APIService %s after %d attempts due to %v", name, attempt, err)
	}
	return
}
