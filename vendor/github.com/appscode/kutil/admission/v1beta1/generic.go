package v1beta1

import (
	"bytes"
	"sync"

	jp "github.com/appscode/jsonpatch"
	"github.com/appscode/kutil/admission"
	"github.com/appscode/kutil/runtime/serializer/versioning"
	"k8s.io/api/admission/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
)

type GenericWebhook struct {
	plural   schema.GroupVersionResource
	singular string

	srcGroups sets.String
	target    schema.GroupVersionKind
	factory   GetterFactory
	get       GetFunc
	handler   admission.ResourceHandler

	initialized bool
	lock        sync.RWMutex
}

var _ AdmissionHook = &GenericWebhook{}

func NewGenericWebhook(
	plural schema.GroupVersionResource,
	singular string,
	srcGroups []string,
	target schema.GroupVersionKind,
	factory GetterFactory,
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
		return StatusUninitialized()
	}

	codec := versioning.NewDefaultingCodecForScheme(
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
			return StatusInternalServerError(err)
		} else if err == nil {
			err2 := h.handler.OnDelete(obj)
			if err2 != nil {
				return StatusBadRequest(err)
			}
		}
	case v1beta1.Create:
		obj, _, err := codec.Decode(req.Object.Raw, &gvk, nil)
		if err != nil {
			return StatusBadRequest(err)
		}

		mod, err := h.handler.OnCreate(obj)
		if err != nil {
			return StatusForbidden(err)
		} else if mod != nil {
			var buf bytes.Buffer
			err = codec.Encode(mod, &buf)
			if err != nil {
				return StatusBadRequest(err)
			}
			ops, err := jp.CreatePatch(req.Object.Raw, buf.Bytes())
			if err != nil {
				return StatusBadRequest(err)
			}
			patch, err := json.Marshal(ops)
			if err != nil {
				return StatusInternalServerError(err)
			}
			status.Patch = patch
			patchType := v1beta1.PatchTypeJSONPatch
			status.PatchType = &patchType
		}
	case v1beta1.Update:
		obj, _, err := codec.Decode(req.Object.Raw, &gvk, nil)
		if err != nil {
			return StatusBadRequest(err)
		}
		oldObj, _, err := codec.Decode(req.OldObject.Raw, &gvk, nil)
		if err != nil {
			return StatusBadRequest(err)
		}

		mod, err := h.handler.OnUpdate(oldObj, obj)
		if err != nil {
			return StatusForbidden(err)
		} else if mod != nil {
			var buf bytes.Buffer
			err = codec.Encode(mod, &buf)
			if err != nil {
				return StatusBadRequest(err)
			}
			ops, err := jp.CreatePatch(req.Object.Raw, buf.Bytes())
			if err != nil {
				return StatusBadRequest(err)
			}
			patch, err := json.Marshal(ops)
			if err != nil {
				return StatusInternalServerError(err)
			}
			status.Patch = patch
			patchType := v1beta1.PatchTypeJSONPatch
			status.PatchType = &patchType
		}
	}

	status.Allowed = true
	return status
}
