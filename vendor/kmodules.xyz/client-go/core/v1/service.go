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

	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
)

func CreateOrPatchService(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Service) *core.Service, opts metav1.PatchOptions) (*core.Service, kutil.VerbType, error) {
	cur, err := c.CoreV1().Services(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		klog.V(3).Infof("Creating Service %s/%s.", meta.Namespace, meta.Name)
		out, err := c.CoreV1().Services(meta.Namespace).Create(ctx, transform(&core.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: core.SchemeGroupVersion.String(),
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
	return PatchService(ctx, c, cur, transform, opts)
}

func PatchService(ctx context.Context, c kubernetes.Interface, cur *core.Service, transform func(*core.Service) *core.Service, opts metav1.PatchOptions) (*core.Service, kutil.VerbType, error) {
	return PatchServiceObject(ctx, c, cur, transform(cur.DeepCopy()), opts)
}

func PatchServiceObject(ctx context.Context, c kubernetes.Interface, cur, mod *core.Service, opts metav1.PatchOptions) (*core.Service, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, core.Service{})
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	klog.V(3).Infof("Patching Service %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.CoreV1().Services(cur.Namespace).Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	return out, kutil.VerbPatched, err
}

func TryUpdateService(ctx context.Context, c kubernetes.Interface, meta metav1.ObjectMeta, transform func(*core.Service) *core.Service, opts metav1.UpdateOptions) (result *core.Service, err error) {
	attempt := 0
	err = wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.RetryTimeout, true, func(ctx context.Context) (bool, error) {
		attempt++
		cur, e2 := c.CoreV1().Services(meta.Namespace).Get(ctx, meta.Name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.CoreV1().Services(cur.Namespace).Update(ctx, transform(cur.DeepCopy()), opts)
			return e2 == nil, nil
		}
		klog.Errorf("Attempt %d failed to update Service %s/%s due to %v.", attempt, cur.Namespace, cur.Name, e2)
		return false, nil
	})
	if err != nil {
		err = errors.Errorf("failed to update Service %s/%s after %d attempts due to %v", meta.Namespace, meta.Name, attempt, err)
	}
	return
}

func MergeServicePorts(cur, desired []core.ServicePort) []core.ServicePort {
	if len(cur) == 0 {
		return desired
	}

	// ports
	curPorts := make(map[string]core.ServicePort)
	for _, p := range cur {
		curPorts[p.Name] = p
	}
	for i, dp := range desired {
		cp, ok := curPorts[dp.Name]

		// svc port not found
		if !ok {
			continue
		}

		if dp.Port == 0 {
			dp.Port = cp.Port
		}
		if dp.NodePort == 0 {
			dp.NodePort = cp.NodePort // avoid reassigning port
		}
		if dp.Protocol == "" {
			dp.Protocol = cp.Protocol
		}
		if dp.AppProtocol == nil {
			dp.AppProtocol = cp.AppProtocol
		}
		desired[i] = dp
	}
	return desired
}

func WaitUntilServiceDeletedBySelector(ctx context.Context, c kubernetes.Interface, namespace string, selector *metav1.LabelSelector) error {
	sel, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return err
	}

	return wait.PollUntilContextTimeout(ctx, kutil.RetryInterval, kutil.ReadinessTimeout, true, func(ctx context.Context) (bool, error) {
		svcList, err := c.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: sel.String(),
		})
		if err != nil {
			return false, nil
		}
		return len(svcList.Items) == 0, nil
	})
}
