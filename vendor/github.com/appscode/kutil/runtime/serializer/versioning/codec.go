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

var serializer = func() runtime.Codec {
	mediaType := "application/json"
	info, ok := runtime.SerializerInfoForMediaType(legacyscheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		panic("unsupported media type " + mediaType)
	}
	return info.Serializer
}()

type codec struct {
	scheme        *runtime.Scheme
	encodeVersion schema.GroupVersion
	decodeVersion schema.GroupVersion
}

// NewDefaultingCodecForScheme is a convenience method for callers that are using a scheme.
func NewDefaultingCodecForScheme(
// TODO: I should be a scheme interface?
	scheme *runtime.Scheme,
	encodeVersion schema.GroupVersion,
	decodeVersion schema.GroupVersion,
) runtime.Codec {
	return codec{
		scheme:        scheme,
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
	c.scheme.Default(out)

	return serializer.Encode(out, w)
}

func (c codec) Decode(data []byte, _ *schema.GroupVersionKind, _ runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	in, gvk, err := serializer.Decode(data, nil, nil)
	if err != nil {
		return nil, gvk, err
	}
	c.scheme.Default(in)

	if gvk.GroupVersion() != c.encodeVersion {
		return nil, gvk, fmt.Errorf("data expected to be of version %s, found %s", c.encodeVersion, gvk)
	}
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
