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

var JSONPrettySerializer = func() *Codec {
	mediaType := "application/json"
	info, ok := runtime.SerializerInfoForMediaType(scheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		panic("unsupported media type " + mediaType)
	}
	return &Codec{&codec{info.PrettySerializer}}
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

// MarshalToPrettyJson marshals an object into pretty json.
func MarshalToPrettyJson(obj runtime.Object, gv schema.GroupVersion) ([]byte, error) {
	encoder := versioning.NewCodec(
		JSONPrettySerializer,
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
// WARNING: `json` tags are not respected when struct converted to map[string]interface{}
// WARNING: Embedded structs are not decoded properly: https://github.com/mitchellh/mapstructure/pull/80
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
		data interface{},
	) (interface{}, error) {
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
