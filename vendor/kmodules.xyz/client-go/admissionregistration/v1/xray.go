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
	"fmt"
	"strings"

	admreg "kmodules.xyz/client-go/admissionregistration"
	apireg_util "kmodules.xyz/client-go/apiregistration/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	"kmodules.xyz/client-go/discovery"
	dynamic_util "kmodules.xyz/client-go/dynamic"
	meta_util "kmodules.xyz/client-go/meta"

	jsonpatch "github.com/evanphx/json-patch"
	v1 "k8s.io/api/admissionregistration/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	apiregistration "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	apireg_cs "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	kutil "kmodules.xyz/client-go"
)

type ValidatingWebhookXray struct {
	config    *rest.Config
	apisvc    string
	testObj   runtime.Object
	op        v1.OperationType
	transform func(_ runtime.Object)
	stopCh    <-chan struct{}
}

func NewCreateValidatingWebhookXray(config *rest.Config, apisvc string, testObj runtime.Object, stopCh <-chan struct{}) *ValidatingWebhookXray {
	return &ValidatingWebhookXray{
		config:    config,
		apisvc:    apisvc,
		testObj:   testObj,
		op:        v1.Create,
		transform: nil,
		stopCh:    stopCh,
	}
}

func NewUpdateValidatingWebhookXray(config *rest.Config, apisvc string, testObj runtime.Object, transform func(_ runtime.Object), stopCh <-chan struct{}) *ValidatingWebhookXray {
	return &ValidatingWebhookXray{
		config:    config,
		apisvc:    apisvc,
		testObj:   testObj,
		op:        v1.Update,
		transform: transform,
		stopCh:    stopCh,
	}
}

func NewDeleteValidatingWebhookXray(config *rest.Config, apisvc string, testObj runtime.Object, transform func(_ runtime.Object), stopCh <-chan struct{}) *ValidatingWebhookXray {
	return &ValidatingWebhookXray{
		config:    config,
		apisvc:    apisvc,
		testObj:   testObj,
		op:        v1.Delete,
		transform: transform,
		stopCh:    stopCh,
	}
}

func retry(err error) error {
	if err == nil ||
		strings.HasPrefix(err.Error(), "Internal error occurred: failed calling admission webhook") ||
		strings.HasPrefix(err.Error(), "Internal error occurred: failed calling webhook") || // https://github.com/kubernetes/kubernetes/pull/70060/files
		kerr.IsNotFound(err) ||
		kerr.IsServiceUnavailable(err) ||
		kerr.IsTimeout(err) ||
		kerr.IsServerTimeout(err) ||
		kerr.IsTooManyRequests(err) {
		return nil
	}
	return err
}

func (d ValidatingWebhookXray) IsActive(ctx context.Context) error {
	kc := kubernetes.NewForConfigOrDie(d.config)
	apireg := apireg_cs.NewForConfigOrDie(d.config)

	if admreg.BypassValidatingWebhookXray() {
		apisvc, err := apireg.ApiregistrationV1().APIServices().Get(ctx, d.apisvc, metav1.GetOptions{})
		if err == nil {
			_ = d.updateAPIService(ctx, apireg, apisvc, nil)
		}
		return nil
	}

	attempt := 0
	var failures []string
	return wait.PollImmediateUntil(kutil.RetryInterval, func() (bool, error) {
		apisvc, err := apireg.ApiregistrationV1().APIServices().Get(ctx, d.apisvc, metav1.GetOptions{})
		if err != nil {
			return false, retry(err)
		}
		for _, cond := range apisvc.Status.Conditions {
			if cond.Type == apiregistration.Available && cond.Status == apiregistration.ConditionTrue {
				// Kubernetes is slow to update APIService.status. So, we double check that the pods are running and ready.
				if apisvc.Spec.Service != nil {
					svc, err := kc.CoreV1().Services(apisvc.Spec.Service.Namespace).Get(ctx, apisvc.Spec.Service.Name, metav1.GetOptions{})
					if err != nil {
						return false, retry(err)
					}

					pods, err := kc.CoreV1().Pods(apisvc.Spec.Service.Namespace).List(ctx, metav1.ListOptions{
						LabelSelector: labels.SelectorFromSet(svc.Spec.Selector).String(),
					})
					if err != nil {
						return false, retry(err)
					}
					if len(pods.Items) == 0 {
						return false, nil
					}
					for _, pod := range pods.Items {
						ready, _ := core_util.PodRunningAndReady(pod)
						if !ready {
							return false, nil
						}
					}
				}
				attempt++
				active, err := d.check(ctx)
				if err != nil {
					failures = append(failures, fmt.Sprintf("Attempt %d to detect ValidatingWebhook activation failed due to %s", attempt, err.Error()))
				}
				err = retry(err)
				if active || err != nil {
					_ = d.updateAPIService(ctx, apireg, apisvc, err)
				}
				if err != nil {
					// log failures only if xray fails, otherwise don't confuse users with intermediate failures.
					for _, msg := range failures {
						klog.Warningln(msg)
					}
				}
				return active, err
			}
		}
		return false, nil
	}, d.stopCh)
}

func (d ValidatingWebhookXray) updateAPIService(ctx context.Context, apireg apireg_cs.Interface, apisvc *apiregistration.APIService, err error) error {
	fn := func(annotations map[string]string) map[string]string {
		if len(annotations) == 0 {
			annotations = map[string]string{}
		}
		if err == nil {
			annotations[admreg.KeyAdmissionWebhookActive] = "true"
			annotations[admreg.KeyAdmissionWebhookStatus] = ""
		} else {
			annotations[admreg.KeyAdmissionWebhookActive] = "false"
			annotations[admreg.KeyAdmissionWebhookStatus] = string(kerr.ReasonForError(err)) + "|" + err.Error()
		}
		return annotations
	}

	_, _, e3 := apireg_util.PatchAPIService(ctx, apireg, apisvc, func(in *apiregistration.APIService) *apiregistration.APIService {
		data, ok := in.Annotations[meta_util.LastAppliedConfigAnnotation]
		if ok {
			u, e2 := runtime.Decode(unstructured.UnstructuredJSONScheme, []byte(data))
			if e2 != nil {
				goto LastAppliedConfig
			}
			m, e2 := meta.Accessor(u)
			if e2 != nil {
				goto LastAppliedConfig
			}
			m.SetAnnotations(fn(m.GetAnnotations()))
			if mod, err := runtime.Encode(unstructured.UnstructuredJSONScheme, u); err == nil {
				in.Annotations[meta_util.LastAppliedConfigAnnotation] = string(mod)
			}
		}

	LastAppliedConfig:
		in.Annotations = fn(in.Annotations)
		return in
	}, metav1.PatchOptions{})
	return e3
}

func (d ValidatingWebhookXray) check(ctx context.Context) (bool, error) {
	kc, err := kubernetes.NewForConfig(d.config)
	if err != nil {
		return false, err
	}

	dc, err := dynamic.NewForConfig(d.config)
	if err != nil {
		return false, err
	}

	gvk := d.testObj.GetObjectKind().GroupVersionKind()
	if gvk.Version == "" {
		return false, admreg.ErrMissingVersion
	}
	if gvk.Kind == "" {
		return false, admreg.ErrMissingKind
	}

	gvr, err := discovery.ResourceForGVK(kc.Discovery(), gvk)
	if err != nil {
		return false, err
	}
	klog.Infof("testing ValidatingWebhook using an object with GVR = %s", gvr.String())

	accessor, err := meta.Accessor(d.testObj)
	if err != nil {
		return false, err
	}

	var ri dynamic.ResourceInterface
	if accessor.GetNamespace() != "" {
		ri = dc.Resource(gvr).Namespace(accessor.GetNamespace())
	} else {
		ri = dc.Resource(gvr)
	}

	objJson, err := json.Marshal(d.testObj)
	if err != nil {
		return false, err
	}

	u := unstructured.Unstructured{}
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(objJson, nil, &u)
	if err != nil {
		return false, err
	}

	if d.op == v1.Create {
		_, err := ri.Create(ctx, &u, metav1.CreateOptions{})
		if kutil.AdmissionWebhookDeniedRequest(err) {
			klog.V(10).Infof("failed to create invalid test object as expected with error: %s", err)
			return true, nil
		} else if err != nil {
			return false, err
		}

		_ = dynamic_util.WaitUntilDeleted(ri, d.stopCh, accessor.GetName())
		return false, admreg.ErrWebhookNotActivated
	} else if d.op == v1.Update {
		_, err := ri.Create(ctx, &u, metav1.CreateOptions{})
		if err != nil {
			return false, err
		}

		mod := d.testObj.DeepCopyObject()
		d.transform(mod)
		modJson, err := json.Marshal(mod)
		if err != nil {
			return false, err
		}

		patch, err := jsonpatch.CreateMergePatch(objJson, modJson)
		if err != nil {
			return false, err
		}

		_, err = ri.Patch(ctx, accessor.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})
		defer func() { _ = dynamic_util.WaitUntilDeleted(ri, d.stopCh, accessor.GetName()) }()

		if kutil.AdmissionWebhookDeniedRequest(err) {
			klog.V(10).Infof("failed to update test object as expected with error: %s", err)
			return true, nil
		} else if err != nil {
			return false, err
		}

		return false, admreg.ErrWebhookNotActivated
	} else if d.op == v1.Delete {
		_, err := ri.Create(ctx, &u, metav1.CreateOptions{})
		if err != nil {
			return false, err
		}

		err = ri.Delete(ctx, accessor.GetName(), metav1.DeleteOptions{})
		if kutil.AdmissionWebhookDeniedRequest(err) {
			defer func() {
				// update to make it valid
				mod := d.testObj.DeepCopyObject()
				d.transform(mod)
				modJson, err := json.Marshal(mod)
				if err != nil {
					return
				}

				patch, err := jsonpatch.CreateMergePatch(objJson, modJson)
				if err != nil {
					return
				}

				_, _ = ri.Patch(ctx, accessor.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})

				// delete
				_ = dynamic_util.WaitUntilDeleted(ri, d.stopCh, accessor.GetName())
			}()

			klog.V(10).Infof("failed to delete test object as expected with error: %s", err)
			return true, nil
		} else if err != nil {
			return false, err
		}
		return false, admreg.ErrWebhookNotActivated
	}

	return false, nil
}
