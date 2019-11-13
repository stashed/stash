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
package generic

import (
	"bytes"
	"sync"

	"kmodules.xyz/client-go/meta"
	"kmodules.xyz/webhook-runtime/admission"
	api "kmodules.xyz/webhook-runtime/admission/v1beta1"
	"kmodules.xyz/webhook-runtime/runtime/serializer/versioning"

	"github.com/golang/glog"
	jsoniter "github.com/json-iterator/go"
	jp "gomodules.xyz/jsonpatch/v2"
	"k8s.io/api/admission/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
)

var json = jsoniter.ConfigFastest

type GenericWebhook struct {
	plural   schema.GroupVersionResource
	singular string

	srcGroups sets.String
	target    schema.GroupVersionKind
	factory   api.GetterFactory
	get       api.GetFunc
	handler   admission.ResourceHandler

	initialized bool
	lock        sync.RWMutex
}

var _ api.AdmissionHook = &GenericWebhook{}

func NewGenericWebhook(
	plural schema.GroupVersionResource,
	singular string,
	srcGroups []string,
	target schema.GroupVersionKind,
	factory api.GetterFactory,
	handler admission.ResourceHandler) *GenericWebhook {
	return &GenericWebhook{
		plural:    plural,
		singular:  singular,
		srcGroups: sets.NewString(srcGroups...),
		target:    target,
		factory:   factory,
		handler:   handler,
	}
}

func (h *GenericWebhook) Resource() (schema.GroupVersionResource, string) {
	return h.plural, h.singular
}

func (h *GenericWebhook) Initialize(config *rest.Config, stopCh <-chan struct{}) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.initialized = true

	var err error
	if h.factory != nil {
		h.get, err = h.factory.New(config)
	}
	return err
}

func (h *GenericWebhook) Admit(req *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	status := &v1beta1.AdmissionResponse{}

	if h.handler == nil ||
		(req.Operation != v1beta1.Create && req.Operation != v1beta1.Update && req.Operation != v1beta1.Delete) ||
		len(req.SubResource) != 0 ||
		!h.srcGroups.Has(req.Kind.Group) ||
		req.Kind.Kind != h.target.Kind {
		status.Allowed = true
		return status
	}

	h.lock.RLock()
	defer h.lock.RUnlock()
	if !h.initialized {
		return api.StatusUninitialized()
	}

	codec := versioning.NewDefaultingCodecForScheme(
		meta.JSONSerializer,
		meta.JSONSerializer,
		legacyscheme.Scheme,
		legacyscheme.Scheme,
		schema.GroupVersion{Group: req.Kind.Group, Version: req.Kind.Version},
		h.target.GroupVersion(),
	)
	gvk := schema.GroupVersionKind{Group: req.Kind.Group, Version: req.Kind.Version, Kind: req.Kind.Kind}

	switch req.Operation {
	case v1beta1.Delete:
		if h.get == nil {
			break
		}
		// req.Object.Raw = nil, so read from kubernetes
		obj, err := h.get(req.Namespace, req.Name)
		if err != nil && !kerr.IsNotFound(err) {
			return api.StatusInternalServerError(err)
		} else if err == nil {
			err2 := h.handler.OnDelete(obj)
			if err2 != nil {
				return api.StatusBadRequest(err)
			}
		}
	case v1beta1.Create:
		obj, _, err := codec.Decode(req.Object.Raw, &gvk, nil)
		if err != nil {
			return api.StatusBadRequest(err)
		}

		mod, err := h.handler.OnCreate(obj)
		if err != nil {
			return api.StatusForbidden(err)
		} else if mod != nil {
			var buf bytes.Buffer
			err = codec.Encode(mod, &buf)
			if err != nil {
				return api.StatusBadRequest(err)
			}
			ops, err := jp.CreatePatch(req.Object.Raw, buf.Bytes())
			if err != nil {
				return api.StatusBadRequest(err)
			}
			patch, err := json.Marshal(ops)
			if err != nil {
				return api.StatusInternalServerError(err)
			}
			if glog.V(8) {
				glog.V(8).Infoln("patch:", string(patch))
			}
			status.Patch = patch
			patchType := v1beta1.PatchTypeJSONPatch
			status.PatchType = &patchType
		}
	case v1beta1.Update:
		obj, _, err := codec.Decode(req.Object.Raw, &gvk, nil)
		if err != nil {
			return api.StatusBadRequest(err)
		}
		oldObj, _, err := codec.Decode(req.OldObject.Raw, &gvk, nil)
		if err != nil {
			return api.StatusBadRequest(err)
		}

		mod, err := h.handler.OnUpdate(oldObj, obj)
		if err != nil {
			return api.StatusForbidden(err)
		} else if mod != nil {
			var buf bytes.Buffer
			err = codec.Encode(mod, &buf)
			if err != nil {
				return api.StatusBadRequest(err)
			}
			ops, err := jp.CreatePatch(req.Object.Raw, buf.Bytes())
			if err != nil {
				return api.StatusBadRequest(err)
			}
			patch, err := json.Marshal(ops)
			if err != nil {
				return api.StatusInternalServerError(err)
			}
			if glog.V(8) {
				glog.V(8).Infoln("patch:", string(patch))
			}
			status.Patch = patch
			patchType := v1beta1.PatchTypeJSONPatch
			status.PatchType = &patchType
		}
	}

	status.Allowed = true
	return status
}
