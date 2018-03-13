package v1beta1

import (
	"sync"

	"github.com/appscode/kutil"
	"github.com/appscode/kutil/admission/api"
	"github.com/appscode/kutil/meta"
	admission "k8s.io/api/admission/v1beta1"
	"k8s.io/api/batch/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
)

type CronJobWebhook struct {
	client   kubernetes.Interface
	handler  api.ResourceHandler
	plural   schema.GroupVersionResource
	singular string

	initialized bool
	lock        sync.RWMutex
}

var _ api.AdmissionHook = &CronJobWebhook{}

func NewCronJobWebhook(plural schema.GroupVersionResource, singular string, handler api.ResourceHandler) *CronJobWebhook {
	return &CronJobWebhook{
		plural:   plural,
		singular: singular,
		handler:  handler,
	}
}

func (a *CronJobWebhook) Resource() (schema.GroupVersionResource, string) {
	return a.plural, a.singular
}

func (a *CronJobWebhook) Initialize(config *rest.Config, stopCh <-chan struct{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.initialized = true

	var err error
	a.client, err = kubernetes.NewForConfig(config)
	return err
}

func (a *CronJobWebhook) Admit(req *admission.AdmissionRequest) *admission.AdmissionResponse {
	status := &admission.AdmissionResponse{}

	if a.handler == nil ||
		(req.Operation != admission.Create && req.Operation != admission.Update && req.Operation != admission.Delete) ||
		len(req.SubResource) != 0 ||
		(req.Kind.Group != v1beta1.GroupName) ||
		req.Kind.Kind != "CronJob" {
		status.Allowed = true
		return status
	}

	a.lock.RLock()
	defer a.lock.RUnlock()
	if !a.initialized {
		return api.StatusUninitialized()
	}
	gv := schema.GroupVersion{Group: req.Kind.Group, Version: req.Kind.Version}

	switch req.Operation {
	case admission.Delete:
		// req.Object.Raw = nil, so read from kubernetes
		obj, err := a.client.BatchV1beta1().CronJobs(req.Namespace).Get(req.Name, metav1.GetOptions{})
		if err != nil && !kerr.IsNotFound(err) {
			return api.StatusInternalServerError(err)
		} else if err == nil {
			err2 := a.handler.OnDelete(obj)
			if err2 != nil {
				return api.StatusBadRequest(err)
			}
		}
	case admission.Create:
		v1beta1Obj, originalObj, err := convert_to_v1beta1_cronjob(gv, req.Object.Raw)
		if err != nil {
			return api.StatusBadRequest(err)
		}

		v1beta1Mod, err := a.handler.OnCreate(v1beta1Obj)
		if err != nil {
			return api.StatusForbidden(err)
		} else if v1beta1Mod != nil {
			patch, err := create_cronjob_patch(gv, originalObj, v1beta1Mod)
			if err != nil {
				return api.StatusInternalServerError(err)
			}
			status.Patch = patch
			patchType := admission.PatchTypeJSONPatch
			status.PatchType = &patchType
		}
	case admission.Update:
		v1beta1Obj, originalObj, err := convert_to_v1beta1_cronjob(gv, req.Object.Raw)
		if err != nil {
			return api.StatusBadRequest(err)
		}
		v1beta1OldObj, _, err := convert_to_v1beta1_cronjob(gv, req.OldObject.Raw)
		if err != nil {
			return api.StatusBadRequest(err)
		}

		v1beta1Mod, err := a.handler.OnUpdate(v1beta1OldObj, v1beta1Obj)
		if err != nil {
			return api.StatusForbidden(err)
		} else if v1beta1Mod != nil {
			patch, err := create_cronjob_patch(gv, originalObj, v1beta1Mod)
			if err != nil {
				return api.StatusInternalServerError(err)
			}
			status.Patch = patch
			patchType := admission.PatchTypeJSONPatch
			status.PatchType = &patchType
		}
	}

	status.Allowed = true
	return status
}

func convert_to_v1beta1_cronjob(gv schema.GroupVersion, raw []byte) (*v1beta1.CronJob, runtime.Object, error) {
	switch gv {
	case v1beta1.SchemeGroupVersion:
		v1beta1Obj := &v1beta1.CronJob{}
		err := json.Unmarshal(raw, v1beta1Obj)
		if err != nil {
			return nil, nil, err
		}
		return v1beta1Obj, v1beta1Obj, nil
	}
	return nil, nil, kutil.ErrUnknown
}

func create_cronjob_patch(gv schema.GroupVersion, originalObj, v1beta1Mod interface{}) ([]byte, error) {
	switch gv {
	case v1beta1.SchemeGroupVersion:
		v1beta1Obj := v1beta1Mod.(runtime.Object)
		legacyscheme.Scheme.Default(v1beta1Obj)
		return meta.CreateJSONPatch(originalObj.(runtime.Object), v1beta1Obj)
	}
	return nil, kutil.ErrUnknown
}
