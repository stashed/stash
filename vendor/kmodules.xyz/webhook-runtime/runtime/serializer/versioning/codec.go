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

package versioning

import (
	"encoding/json"
	"io"
	"sync"

	_ "kmodules.xyz/openshift/apis/apps/install"

	_ "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/klog"
	_ "k8s.io/kubernetes/pkg/apis/apps/install"
	_ "k8s.io/kubernetes/pkg/apis/batch/install"
	_ "k8s.io/kubernetes/pkg/apis/core/install"
	_ "k8s.io/kubernetes/pkg/apis/extensions/install"
	_ "k8s.io/kubernetes/pkg/apis/rbac/install"
	_ "k8s.io/kubernetes/pkg/apis/storage/install"
)

type codec struct {
	encoder       runtime.Encoder
	decoder       runtime.Decoder
	scheme        *runtime.Scheme
	defaulter     runtime.ObjectDefaulter
	encodeVersion runtime.GroupVersioner
	decodeVersion runtime.GroupVersioner

	identifier runtime.Identifier
}

// ref: https://github.com/kubernetes/apimachinery/blob/v0.18.3/pkg/runtime/serializer/versioning/versioning.go#L92-L121
var identifiersMap sync.Map

type codecIdentifier struct {
	EncodeGV string `json:"encodeGV,omitempty"`
	Encoder  string `json:"encoder,omitempty"`
	Name     string `json:"name,omitempty"`
}

// identifier computes Identifier of Encoder based on codec parameters.
func identifier(encodeGV runtime.GroupVersioner, encoder runtime.Encoder) runtime.Identifier {
	result := codecIdentifier{
		Name: "webhook-versioning",
	}

	if encodeGV != nil {
		result.EncodeGV = encodeGV.Identifier()
	}
	if encoder != nil {
		result.Encoder = string(encoder.Identifier())
	}
	if id, ok := identifiersMap.Load(result); ok {
		return id.(runtime.Identifier)
	}
	identifier, err := json.Marshal(result)
	if err != nil {
		klog.Fatalf("Failed marshaling identifier for codec: %v", err)
	}
	identifiersMap.Store(result, runtime.Identifier(identifier))
	return runtime.Identifier(identifier)
}

// NewDefaultingCodecForScheme is a convenience method for callers that are using a scheme.
func NewDefaultingCodecForScheme(
	encoder runtime.Encoder,
	decoder runtime.Decoder,
	scheme *runtime.Scheme,
	defaulter runtime.ObjectDefaulter,
	encodeVersion runtime.GroupVersioner,
	decodeVersion runtime.GroupVersioner,
) runtime.Codec {
	return codec{
		encoder:       encoder,
		decoder:       decoder,
		scheme:        scheme,
		defaulter:     defaulter,
		encodeVersion: encodeVersion,
		decodeVersion: decodeVersion,
		identifier:    identifier(encodeVersion, encoder),
	}
}

func (c codec) Identifier() runtime.Identifier {
	return c.identifier
}

func (c codec) Encode(obj runtime.Object, w io.Writer) error {
	if co, ok := obj.(runtime.CacheableObject); ok {
		return co.CacheEncode(c.Identifier(), c.doEncode, w)
	}
	return c.doEncode(obj, w)
}

func (c *codec) doEncode(obj runtime.Object, w io.Writer) error {
	var out runtime.Object

	kinds, isUnversioned, err := c.scheme.ObjectKinds(obj)
	if err != nil {
		return err
	}

	if isUnversioned {
		out = obj
	} else {
		// ref: k8s.io/apimachinery/pkg/runtime/scheme.go
		target, ok := c.encodeVersion.KindForGroupVersionKinds(kinds)
		if ok {
			// target wants to use the existing type, set kind and return (no conversion necessary)
			for _, kind := range kinds {
				if target == kind {
					obj.GetObjectKind().SetGroupVersionKind(kind)
					out = obj
					break
				}
			}
		}

		if out == nil {
			internal, err := c.scheme.UnsafeConvertToVersion(obj, runtime.InternalGroupVersioner)
			if err != nil {
				return err
			}

			out, err = c.scheme.UnsafeConvertToVersion(internal, c.encodeVersion)
			if err != nil {
				return err
			}
		}
	}

	if c.defaulter != nil {
		c.defaulter.Default(out)
	}

	return c.encoder.Encode(out, w)
}

func (c codec) Decode(data []byte, gvk *schema.GroupVersionKind, _ runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	in, gvk, err := c.decoder.Decode(data, gvk, nil)
	if err != nil {
		return nil, gvk, err
	}

	if c.defaulter != nil {
		c.defaulter.Default(in)
	}
	in.GetObjectKind().SetGroupVersionKind(*gvk)

	if target, ok := c.decodeVersion.KindForGroupVersionKinds([]schema.GroupVersionKind{*gvk}); ok && target == *gvk {
		return in, gvk, err
	}

	internal, err := c.scheme.UnsafeConvertToVersion(in, runtime.InternalGroupVersioner)
	if err != nil {
		return nil, gvk, err
	}

	out, err := c.scheme.UnsafeConvertToVersion(internal, c.decodeVersion)
	return out, gvk, err
}
