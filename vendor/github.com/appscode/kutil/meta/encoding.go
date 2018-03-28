package meta

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/versioning"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
)

// MarshalToYAML marshals an object into yaml.
func MarshalToYAML(obj runtime.Object, gv schema.GroupVersion) ([]byte, error) {
	mediaType := "application/yaml"
	info, ok := runtime.SerializerInfoForMediaType(clientsetscheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return []byte{}, errors.Errorf("unsupported media type %q", mediaType)
	}

	encoder := versioning.NewCodecForScheme(
		clientsetscheme.Scheme,
		info.Serializer,
		nil,
		gv,
		nil,
	)
	return runtime.Encode(encoder, obj)
}

// UnmarshalFromYAML unmarshals an object into yaml.
func UnmarshalFromYAML(data []byte, gv schema.GroupVersion) (runtime.Object, error) {
	mediaType := "application/yaml"
	info, ok := runtime.SerializerInfoForMediaType(clientsetscheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return nil, errors.Errorf("unsupported media type %q", mediaType)
	}

	decoder := versioning.NewCodecForScheme(
		clientsetscheme.Scheme,
		nil,
		info.Serializer,
		nil,
		gv,
	)
	return runtime.Decode(decoder, data)
}

// MarshalToJson marshals an object into json.
func MarshalToJson(obj runtime.Object, gv schema.GroupVersion) ([]byte, error) {
	mediaType := "application/json"
	info, ok := runtime.SerializerInfoForMediaType(clientsetscheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return []byte{}, errors.Errorf("unsupported media type %q", mediaType)
	}

	encoder := versioning.NewCodecForScheme(
		clientsetscheme.Scheme,
		info.Serializer,
		nil,
		gv,
		nil,
	)
	return runtime.Encode(encoder, obj)
}

// UnmarshalFromJSON unmarshals an object into json.
func UnmarshalFromJSON(data []byte, gv schema.GroupVersion) (runtime.Object, error) {
	mediaType := "application/json"
	info, ok := runtime.SerializerInfoForMediaType(clientsetscheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return nil, errors.Errorf("unsupported media type %q", mediaType)
	}

	decoder := versioning.NewCodecForScheme(
		clientsetscheme.Scheme,
		nil,
		info.Serializer,
		nil,
		gv,
	)
	return runtime.Decode(decoder, data)
}
