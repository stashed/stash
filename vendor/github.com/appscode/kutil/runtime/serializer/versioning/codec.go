package versioning

import (
	"fmt"
	"io"

	_ "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	_ "k8s.io/kubernetes/pkg/apis/apps/install"
	_ "k8s.io/kubernetes/pkg/apis/batch/install"
	_ "k8s.io/kubernetes/pkg/apis/core/install"
	_ "k8s.io/kubernetes/pkg/apis/extensions/install"
)

var JSONSerializer = func() runtime.Codec {
	mediaType := "application/json"
	info, ok := runtime.SerializerInfoForMediaType(legacyscheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		panic("unsupported media type " + mediaType)
	}
	return info.Serializer
}()

var YAMLSerializer = func() runtime.Codec {
	mediaType := "application/yaml"
	info, ok := runtime.SerializerInfoForMediaType(legacyscheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		panic("unsupported media type " + mediaType)
	}
	return info.Serializer
}()

type codec struct {
	encoder runtime.Encoder
	decoder runtime.Decoder
	scheme        *runtime.Scheme
	defaulter     runtime.ObjectDefaulter
	encodeVersion schema.GroupVersion
	decodeVersion schema.GroupVersion
}

// NewDefaultingCodecForScheme is a convenience method for callers that are using a scheme.
func NewDefaultingCodecForScheme(
	encoder runtime.Encoder,
	decoder runtime.Decoder,
	scheme *runtime.Scheme,
	defaulter runtime.ObjectDefaulter,
	encodeVersion schema.GroupVersion,
	decodeVersion schema.GroupVersion,
) runtime.Codec {
	return codec{
		encoder:   encoder,
		decoder:   decoder,
		scheme:        scheme,
		defaulter:     defaulter,
		encodeVersion: encodeVersion,
		decodeVersion: decodeVersion,
	}
}

func (c codec) Encode(obj runtime.Object, w io.Writer) error {
	var out runtime.Object
	if c.encodeVersion == c.decodeVersion {
		out = obj
	} else {
		internal, err := c.scheme.UnsafeConvertToVersion(obj, runtime.InternalGroupVersioner)
		if err != nil {
			return err
		}

		out, err = c.scheme.UnsafeConvertToVersion(internal, c.encodeVersion)
		if err != nil {
			return err
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
	if gvk.GroupVersion() != c.encodeVersion {
		return nil, gvk, fmt.Errorf("data expected to be of version %s, found %s", c.encodeVersion, gvk)
	}

	if c.defaulter != nil {
		c.defaulter.Default(in)
	}
	in.GetObjectKind().SetGroupVersionKind(*gvk)

	if c.encodeVersion == c.decodeVersion {
		return in, gvk, err
	}

	internal, err := c.scheme.UnsafeConvertToVersion(in, runtime.InternalGroupVersioner)
	if err != nil {
		return nil, gvk, err
	}

	out, err := c.scheme.UnsafeConvertToVersion(internal, c.decodeVersion)
	return out, gvk, err
}
