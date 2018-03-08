package v1beta1

import (
	"sync"

	"github.com/appscode/kutil"
	"github.com/appscode/kutil/admission/api"
	"github.com/appscode/kutil/meta"
	admission "k8s.io/api/admission/v1beta1"
	"k8s.io/api/apps/v1"
	"k8s.io/api/apps/v1beta2"
	extensions "k8s.io/api/extensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
)

type DaemonSetWebhook struct {
	client   kubernetes.Interface
	handler  api.ResourceHandler
	plural   schema.GroupVersionResource
	singular string

	initialized bool
	lock        sync.RWMutex
}

var _ api.AdmissionHook = &DaemonSetWebhook{}

func NewDaemonSetWebhook(plural schema.GroupVersionResource, singular string, handler api.ResourceHandler) *DaemonSetWebhook {
	return &DaemonSetWebhook{
		plural:   plural,
		singular: singular,
		handler:  handler,
	}
}

func (a *DaemonSetWebhook) Resource() (schema.GroupVersionResource, string) {
	return a.plural, a.singular
}

func (a *DaemonSetWebhook) Initialize(config *rest.Config, stopCh <-chan struct{}) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.initialized = true

	var err error
	a.client, err = kubernetes.NewForConfig(config)
	return err
}

func (a *DaemonSetWebhook) Admit(req *admission.AdmissionRequest) *admission.AdmissionResponse {
	status := &admission.AdmissionResponse{}

	if a.handler == nil ||
		(req.Operation != admission.Create && req.Operation != admission.Update && req.Operation != admission.Delete) ||
		len(req.SubResource) != 0 ||
		(req.Kind.Group != v1.GroupName && req.Kind.Group != extensions.GroupName) ||
		req.Kind.Kind != "DaemonSet" {
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
		obj, err := a.client.AppsV1().DaemonSets(req.Namespace).Get(req.Name, metav1.GetOptions{})
		if err != nil && !kerr.IsNotFound(err) {
			return api.StatusInternalServerError(err)
		} else if err == nil {
			err2 := a.handler.OnDelete(obj)
			if err2 != nil {
				return api.StatusBadRequest(err)
			}
		}
	case admission.Create:
		extObj, originalObj, err := convert_to_extensions_daemonset(gv, req.Object.Raw)
		if err != nil {
			return api.StatusBadRequest(err)
		}

		extMod, err := a.handler.OnCreate(extObj)
		if err != nil {
			return api.StatusForbidden(err)
		} else if extMod != nil {
			patch, err := create_daemonset_patch(gv, originalObj, extMod)
			if err != nil {
				return api.StatusInternalServerError(err)
			}
			status.Patch = patch
			patchType := admission.PatchTypeJSONPatch
			status.PatchType = &patchType
		}
	case admission.Update:
		extObj, originalObj, err := convert_to_extensions_daemonset(gv, req.Object.Raw)
		if err != nil {
			return api.StatusBadRequest(err)
		}
		extOldObj, _, err := convert_to_extensions_daemonset(gv, req.OldObject.Raw)
		if err != nil {
			return api.StatusBadRequest(err)
		}

		extMod, err := a.handler.OnUpdate(extOldObj, extObj)
		if err != nil {
			return api.StatusForbidden(err)
		} else if extMod != nil {
			patch, err := create_daemonset_patch(gv, originalObj, extMod)
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

func convert_to_extensions_daemonset(gv schema.GroupVersion, raw []byte) (*extensions.DaemonSet, runtime.Object, error) {
	switch gv {
	case v1.SchemeGroupVersion:
		v1Obj := &v1.DaemonSet{}
		err := json.Unmarshal(raw, v1Obj)
		if err != nil {
			return nil, nil, err
		}

		extObj := &extensions.DaemonSet{}
		err = scheme.Scheme.Convert(v1Obj, extObj, nil)
		if err != nil {
			return nil, nil, err
		}
		return extObj, v1Obj, nil

	case v1beta2.SchemeGroupVersion:
		v1beta2Obj := &v1beta2.DaemonSet{}
		err := json.Unmarshal(raw, v1beta2Obj)
		if err != nil {
			return nil, nil, err
		}

		extObj := &extensions.DaemonSet{}
		err = scheme.Scheme.Convert(v1beta2Obj, extObj, nil)
		if err != nil {
			return nil, nil, err
		}
		return extObj, v1beta2Obj, nil

	case extensions.SchemeGroupVersion:
		extObj := &extensions.DaemonSet{}
		err := json.Unmarshal(raw, extObj)
		if err != nil {
			return nil, nil, err
		}
		return extObj, extObj, nil
	}
	return nil, nil, kutil.ErrUnknown
}

func create_daemonset_patch(gv schema.GroupVersion, originalObj, extMod interface{}) ([]byte, error) {
	switch gv {
	case v1.SchemeGroupVersion:
		v1Mod := &v1.DaemonSet{}
		err := scheme.Scheme.Convert(extMod, v1Mod, nil)
		if err != nil {
			return nil, err
		}
		legacyscheme.Scheme.Default(v1Mod)
		return meta.CreateJSONPatch(originalObj.(runtime.Object), v1Mod)

	case v1beta2.SchemeGroupVersion:
		v1beta2Mod := &v1beta2.DaemonSet{}
		err := scheme.Scheme.Convert(extMod, v1beta2Mod, nil)
		if err != nil {
			return nil, err
		}
		legacyscheme.Scheme.Default(v1beta2Mod)
		return meta.CreateJSONPatch(originalObj.(runtime.Object), v1beta2Mod)

	case extensions.SchemeGroupVersion:
		extObj := extMod.(runtime.Object)
		legacyscheme.Scheme.Default(extObj)
		return meta.CreateJSONPatch(originalObj.(runtime.Object), extObj)
	}
	return nil, kutil.ErrUnknown
}
