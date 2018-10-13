package v1beta1

import (
	"fmt"
	"strings"

	"github.com/appscode/kutil"
	apireg_util "github.com/appscode/kutil/apiregistration/v1beta1"
	core_util "github.com/appscode/kutil/core/v1"
	"github.com/appscode/kutil/discovery"
	meta_util "github.com/appscode/kutil/meta"
	"github.com/evanphx/json-patch"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"k8s.io/api/admissionregistration/v1beta1"
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
	apiregistration "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	apireg_cs "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

func init() {
	pflag.BoolVar(&bypassValidatingWebhookXray, "bypass-validating-webhook-xray", bypassValidatingWebhookXray, "if true, bypasses validating webhook xray checks")
}

const (
	KeyAdmissionWebhookActive = "admission-webhook.appscode.com/active"
	KeyAdmissionWebhookStatus = "admission-webhook.appscode.com/status"
)

var bypassValidatingWebhookXray = false

var ErrMissingKind = errors.New("test object missing kind")
var ErrMissingVersion = errors.New("test object missing version")
var ErrWebhookNotActivated = errors.New("Admission webhooks are not activated. Enable it by configuring --enable-admission-plugins flag of kube-apiserver. For details, visit: https://appsco.de/kube-apiserver-webhooks")

type ValidatingWebhookXray struct {
	config    *rest.Config
	apisvc    string
	webhook   string
	testObj   runtime.Object
	op        v1beta1.OperationType
	transform func(_ runtime.Object)
	stopCh    <-chan struct{}
}

func NewCreateValidatingWebhookXray(config *rest.Config, apisvc, webhook string, testObj runtime.Object, stopCh <-chan struct{}) *ValidatingWebhookXray {
	return &ValidatingWebhookXray{
		config:    config,
		apisvc:    apisvc,
		webhook:   webhook,
		testObj:   testObj,
		op:        v1beta1.Create,
		transform: nil,
		stopCh:    stopCh,
	}
}

func NewUpdateValidatingWebhookXray(config *rest.Config, apisvc, webhook string, testObj runtime.Object, transform func(_ runtime.Object), stopCh <-chan struct{}) *ValidatingWebhookXray {
	return &ValidatingWebhookXray{
		config:    config,
		apisvc:    apisvc,
		webhook:   webhook,
		testObj:   testObj,
		op:        v1beta1.Update,
		transform: transform,
		stopCh:    stopCh,
	}
}

func NewDeleteValidatingWebhookXray(config *rest.Config, apisvc, webhook string, testObj runtime.Object, transform func(_ runtime.Object), stopCh <-chan struct{}) *ValidatingWebhookXray {
	return &ValidatingWebhookXray{
		config:    config,
		apisvc:    apisvc,
		webhook:   webhook,
		testObj:   testObj,
		op:        v1beta1.Delete,
		transform: transform,
		stopCh:    stopCh,
	}
}

func retry(err error) error {
	if err == nil {
		return nil
	}
	if strings.HasPrefix(err.Error(), "Internal error occurred: failed calling admission webhook") ||
		kerr.IsNotFound(err) ||
		kerr.IsServiceUnavailable(err) ||
		kerr.IsTimeout(err) ||
		kerr.IsServerTimeout(err) ||
		kerr.IsTooManyRequests(err) {
		return nil
	}
	return err
}

func (d ValidatingWebhookXray) IsActive() error {
	kc := kubernetes.NewForConfigOrDie(d.config)
	apireg := apireg_cs.NewForConfigOrDie(d.config)

	if bypassValidatingWebhookXray {
		apisvc, err := apireg.ApiregistrationV1beta1().APIServices().Get(d.apisvc, metav1.GetOptions{})
		if err == nil {
			d.updateAPIService(apireg, apisvc, nil)
		}
		return nil
	}

	attempt := 0
	return wait.PollImmediateUntil(kutil.RetryInterval, func() (bool, error) {
		apisvc, err := apireg.ApiregistrationV1beta1().APIServices().Get(d.apisvc, metav1.GetOptions{})
		if err != nil {
			return false, retry(err)
		}
		for _, cond := range apisvc.Status.Conditions {
			if cond.Type == apiregistration.Available && cond.Status == apiregistration.ConditionTrue {
				// Kubernetes is slow to update APIService.status. So, we double check that the pods are running and ready.
				if apisvc.Spec.Service != nil {
					svc, err := kc.CoreV1().Services(apisvc.Spec.Service.Namespace).Get(apisvc.Spec.Service.Name, metav1.GetOptions{})
					if err != nil {
						return false, retry(err)
					}

					pods, err := kc.CoreV1().Pods(apisvc.Spec.Service.Namespace).List(metav1.ListOptions{
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
				active, err := d.check()
				if err != nil {
					glog.Warningf("Attempt %d to detect ValidatingWebhook activation failed due to %s", attempt, err.Error())
				}
				err = retry(err)
				if active || err != nil {
					d.updateAPIService(apireg, apisvc, err)
				}
				return active, err
			}
		}
		return false, nil
	}, d.stopCh)
}

func (d ValidatingWebhookXray) updateAPIService(apireg apireg_cs.Interface, apisvc *apiregistration.APIService, err error) {
	fn := func(annotations map[string]string) map[string]string {
		if len(annotations) == 0 {
			annotations = map[string]string{}
		}
		if err == nil {
			annotations[KeyAdmissionWebhookActive] = "true"
			annotations[KeyAdmissionWebhookStatus] = ""
		} else {
			annotations[KeyAdmissionWebhookActive] = "false"
			annotations[KeyAdmissionWebhookStatus] = err.Error()
		}
		return annotations
	}

	apireg_util.PatchAPIService(apireg, apisvc, func(in *apiregistration.APIService) *apiregistration.APIService {
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
	})
}

func (d ValidatingWebhookXray) errForbidden() string {
	return fmt.Sprintf(`admission webhook %q denied the request:`, d.webhook)
}

func (d ValidatingWebhookXray) check() (bool, error) {
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
		return false, ErrMissingVersion
	}
	if gvk.Kind == "" {
		return false, ErrMissingKind
	}

	gvr, err := discovery.ResourceForGVK(kc.Discovery(), gvk)
	if err != nil {
		return false, err
	}
	glog.Infof("testing ValidatingWebhook %s using an object with GVR = %s", d.webhook, gvr.String())

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

	if d.op == v1beta1.Create {
		_, err := ri.Create(&u, metav1.CreateOptions{})
		if kerr.IsForbidden(err) &&
			strings.HasPrefix(err.Error(), d.errForbidden()) {
			glog.Infof("failed to create invalid test object as expected with error: %s", err)
			return true, nil
		} else if err != nil {
			return false, err
		}

		ri.Delete(accessor.GetName(), &metav1.DeleteOptions{})
		return false, ErrWebhookNotActivated
	} else if d.op == v1beta1.Update {
		_, err := ri.Create(&u, metav1.CreateOptions{})
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

		_, err = ri.Patch(accessor.GetName(), types.MergePatchType, patch, metav1.UpdateOptions{})
		defer ri.Delete(accessor.GetName(), &metav1.DeleteOptions{})

		if kerr.IsForbidden(err) &&
			strings.HasPrefix(err.Error(), d.errForbidden()) {
			glog.Infof("failed to update test object as expected with error: %s", err)
			return true, nil
		} else if err != nil {
			return false, err
		}

		return false, ErrWebhookNotActivated
	} else if d.op == v1beta1.Delete {
		_, err := ri.Create(&u, metav1.CreateOptions{})
		if err != nil {
			return false, err
		}

		err = ri.Delete(accessor.GetName(), &metav1.DeleteOptions{})
		if kerr.IsForbidden(err) &&
			strings.HasPrefix(err.Error(), d.errForbidden()) {
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

				ri.Patch(accessor.GetName(), types.MergePatchType, patch, metav1.UpdateOptions{})

				// delete
				ri.Delete(accessor.GetName(), &metav1.DeleteOptions{})
			}()

			glog.Infof("failed to delete test object as expected with error: %s", err)
			return true, nil
		} else if err != nil {
			return false, err
		}
		return false, ErrWebhookNotActivated
	}

	return false, nil
}
