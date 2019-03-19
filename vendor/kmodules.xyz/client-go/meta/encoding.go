package meta

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/versioning"
	"k8s.io/client-go/kubernetes/scheme"
)

type codec struct {
	runtime.Codec
}

type Codec struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*codec
}

var JSONSerializer = func() *Codec {
	mediaType := "application/json"
	info, ok := runtime.SerializerInfoForMediaType(scheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		panic("unsupported media type " + mediaType)
	}
	return &Codec{&codec{info.Serializer}}
}()

var YAMLSerializer = func() *Codec {
	mediaType := "application/yaml"
	info, ok := runtime.SerializerInfoForMediaType(scheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		panic("unsupported media type " + mediaType)
	}
	return &Codec{&codec{info.Serializer}}
}()

// MarshalToYAML marshals an object into yaml.
func MarshalToYAML(obj runtime.Object, gv schema.GroupVersion) ([]byte, error) {
	encoder := versioning.NewCodec(
		YAMLSerializer,
		nil,
		runtime.UnsafeObjectConvertor(scheme.Scheme),
		scheme.Scheme,
		scheme.Scheme,
		nil,
		gv,
		nil,
		scheme.Scheme.Name(),
	)
	return runtime.Encode(encoder, obj)
}

// UnmarshalFromYAML unmarshals an object into yaml.
func UnmarshalFromYAML(data []byte, gv schema.GroupVersion) (runtime.Object, error) {
	decoder := versioning.NewCodec(
		nil,
		YAMLSerializer,
		runtime.UnsafeObjectConvertor(scheme.Scheme),
		scheme.Scheme,
		scheme.Scheme,
		nil,
		nil,
		gv,
		scheme.Scheme.Name(),
	)
	return runtime.Decode(decoder, data)
}

// MarshalToJson marshals an object into json.
func MarshalToJson(obj runtime.Object, gv schema.GroupVersion) ([]byte, error) {
	encoder := versioning.NewCodec(
		JSONSerializer,
		nil,
		runtime.UnsafeObjectConvertor(scheme.Scheme),
		scheme.Scheme,
		scheme.Scheme,
		nil,
		gv,
		nil,
		scheme.Scheme.Name(),
	)
	return runtime.Encode(encoder, obj)
}

// UnmarshalFromJSON unmarshals an object into json.
func UnmarshalFromJSON(data []byte, gv schema.GroupVersion) (runtime.Object, error) {
	decoder := versioning.NewCodec(
		nil,
		JSONSerializer,
		runtime.UnsafeObjectConvertor(scheme.Scheme),
		scheme.Scheme,
		scheme.Scheme,
		nil,
		nil,
		gv,
		scheme.Scheme.Name(),
	)
	return runtime.Decode(decoder, data)
}

// Decode takes an input structure and uses reflection to translate it to
// the output structure. output must be a pointer to a map or struct.
//
// WARNING: Embedded structs are not decoded properly: https://github.com/mitchellh/mapstructure/pull/80
//
func Decode(input interface{}, output interface{}) error {
	config := &mapstructure.DecoderConfig{
		DecodeHook: StringToQuantityHookFunc(),
		Metadata:   nil,
		Result:     output,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	return decoder.Decode(input)
}

// StringToQuantityHookFunc returns a DecodeHookFunc that converts string to resource.Quantity
func StringToQuantityHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(resource.Quantity{}) {
			return data, nil
		}

		// Convert it by parsing
		return resource.ParseQuantity(data.(string))
	}
}
