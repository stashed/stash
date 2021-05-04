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
	"fmt"

	"github.com/pkg/errors"
	reg "k8s.io/api/admissionregistration/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
)

func CreateOrPatchMutatingWebhookConfiguration(ctx context.Context, c kubernetes.Interface, name string, transform func(*reg.MutatingWebhookConfiguration) *reg.MutatingWebhookConfiguration, opts metav1.PatchOptions) (*reg.MutatingWebhookConfiguration, kutil.VerbType, error) {
	cur, err := c.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		klog.V(3).Infof("Creating MutatingWebhookConfiguration %s.", name)
		out, err := c.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(ctx, transform(&reg.MutatingWebhookConfiguration{
			TypeMeta: metav1.TypeMeta{
				Kind:       "MutatingWebhookConfiguration",
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
	return PatchMutatingWebhookConfiguration(ctx, c, cur, transform, opts)
}

func PatchMutatingWebhookConfiguration(ctx context.Context, c kubernetes.Interface, cur *reg.MutatingWebhookConfiguration, transform func(*reg.MutatingWebhookConfiguration) *reg.MutatingWebhookConfiguration, opts metav1.PatchOptions) (*reg.MutatingWebhookConfiguration, kutil.VerbType, error) {
	return PatchMutatingWebhookConfigurationObject(ctx, c, cur, transform(cur.DeepCopy()), opts)
}

func PatchMutatingWebhookConfigurationObject(ctx context.Context, c kubernetes.Interface, cur, mod *reg.MutatingWebhookConfiguration, opts metav1.PatchOptions) (*reg.MutatingWebhookConfiguration, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(mod)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, reg.MutatingWebhookConfiguration{})
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	klog.V(3).Infof("Patching MutatingWebhookConfiguration %s with %s.", cur.Name, string(patch))
	out, err := c.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Patch(ctx, cur.Name, types.StrategicMergePatchType, patch, opts)
	return out, kutil.VerbPatched, err
}

func TryUpdateMutatingWebhookConfiguration(ctx context.Context, c kubernetes.Interface, name string, transform func(*reg.MutatingWebhookConfiguration) *reg.MutatingWebhookConfiguration, opts metav1.UpdateOptions) (result *reg.MutatingWebhookConfiguration, err error) {
	attempt := 0
	err = wait.PollImmediate(kutil.RetryInterval, kutil.RetryTimeout, func() (bool, error) {
		attempt++
		cur, e2 := c.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(ctx, name, metav1.GetOptions{})
		if kerr.IsNotFound(e2) {
			return false, e2
		} else if e2 == nil {
			result, e2 = c.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Update(ctx, transform(cur.DeepCopy()), opts)
			return e2 == nil, nil
		}
		klog.Errorf("Attempt %d failed to update MutatingWebhookConfiguration %s due to %v.", attempt, cur.Name, e2)
		return false, nil
	})

	if err != nil {
		err = errors.Errorf("failed to update MutatingWebhookConfiguration %s after %d attempts due to %v", name, attempt, err)
	}
	return
}

func UpdateMutatingWebhookCABundle(config *rest.Config, webhookConfigName string, extraConditions ...watchtools.ConditionFunc) error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, kutil.ReadinessTimeout)
	defer cancel()

	err := rest.LoadTLSFiles(config)
	if err != nil {
		return err
	}

	kc := kubernetes.NewForConfigOrDie(config)
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fields.OneTermEqualSelector(kutil.ObjectNameField, webhookConfigName).String()
			return kc.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fields.OneTermEqualSelector(kutil.ObjectNameField, webhookConfigName).String()
			return kc.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Watch(ctx, options)
		},
	}

	var conditions = append([]watchtools.ConditionFunc{
		func(event watch.Event) (bool, error) {
			switch event.Type {
			case watch.Deleted:
				return false, nil
			case watch.Error:
				return false, errors.New("error watching")
			case watch.Added, watch.Modified:
				cur := event.Object.(*reg.MutatingWebhookConfiguration)
				_, _, err := PatchMutatingWebhookConfiguration(context.TODO(), kc, cur, func(in *reg.MutatingWebhookConfiguration) *reg.MutatingWebhookConfiguration {
					for i := range in.Webhooks {
						in.Webhooks[i].ClientConfig.CABundle = config.CAData
					}
					return in
				}, metav1.PatchOptions{})
				if err != nil {
					klog.Warning(err)
				}
				return err == nil, err
			default:
				return false, fmt.Errorf("unexpected event type: %v", event.Type)
			}
		},
	}, extraConditions...)

	_, err = watchtools.UntilWithSync(
		ctx,
		lw,
		&reg.MutatingWebhookConfiguration{},
		nil,
		conditions...)
	return err
}

func SyncMutatingWebhookCABundle(config *rest.Config, webhookConfigName string) (cancel context.CancelFunc, err error) {
	ctx := context.Background()
	ctx, cancel = context.WithCancel(ctx)

	err = rest.LoadTLSFiles(config)
	if err != nil {
		return
	}

	kc := kubernetes.NewForConfigOrDie(config)
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fields.OneTermEqualSelector(kutil.ObjectNameField, webhookConfigName).String()
			return kc.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fields.OneTermEqualSelector(kutil.ObjectNameField, webhookConfigName).String()
			return kc.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Watch(ctx, options)
		},
	}

	go func() {
		_, err := watchtools.UntilWithSync(
			ctx,
			lw,
			&reg.MutatingWebhookConfiguration{},
			nil,
			func(event watch.Event) (bool, error) {
				switch event.Type {
				case watch.Deleted:
					return false, nil
				case watch.Error:
					return false, errors.New("error watching")
				case watch.Added, watch.Modified:
					cur := event.Object.(*reg.MutatingWebhookConfiguration)
					_, _, err := PatchMutatingWebhookConfiguration(context.TODO(), kc, cur, func(in *reg.MutatingWebhookConfiguration) *reg.MutatingWebhookConfiguration {
						for i := range in.Webhooks {
							in.Webhooks[i].ClientConfig.CABundle = config.CAData
						}
						return in
					}, metav1.PatchOptions{})
					if err != nil {
						klog.Warning(err)
					}
					return false, nil // continue
				default:
					return false, fmt.Errorf("unexpected event type: %v", event.Type)
				}
			})
		utilruntime.Must(err)
	}()
	return
}
