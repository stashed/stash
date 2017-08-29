package clientset

import (
	"encoding/json"
	"io"
	"net/url"
	"reflect"
	"strings"

	"github.com/appscode/log"
	sapi "github.com/appscode/stash/api"
	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubejson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/pkg/api"
)

// TODO(@sadlil): Find a better way to replace ExtendedCodec to encode and decode objects.
// Follow the guide to replace it with apiv1.Codec and apiv1.ParameterCodecs.
var ExtendedCodec = &extendedCodec{}

// DirectCodecFactory provides methods for retrieving "DirectCodec"s, which do not do conversion.
type DirectCodecFactory struct {
	*extendedCodec
}

// EncoderForVersion returns an encoder that does not do conversion. gv is ignored.
func (f DirectCodecFactory) EncoderForVersion(serializer runtime.Encoder, _ runtime.GroupVersioner) runtime.Encoder {
	return serializer
}

// DecoderToVersion returns an decoder that does not do conversion. gv is ignored.
func (f DirectCodecFactory) DecoderToVersion(serializer runtime.Decoder, _ runtime.GroupVersioner) runtime.Decoder {
	return serializer
}

// SupportedMediaTypes returns the RFC2046 media types that this factory has serializers for.
func (f DirectCodecFactory) SupportedMediaTypes() []runtime.SerializerInfo {
	return []runtime.SerializerInfo{
		{
			MediaType:        "application/json",
			EncodesAsText:    true,
			Serializer:       &extendedCodec{},
			PrettySerializer: &extendedCodec{pretty: true},
			StreamSerializer: &runtime.StreamSerializerInfo{
				Framer:        kubejson.Framer,
				EncodesAsText: true,
				Serializer:    &extendedCodec{},
			},
		},
		{
			MediaType:        "application/yaml",
			EncodesAsText:    true,
			Serializer:       &extendedCodec{yaml: true},
			PrettySerializer: &extendedCodec{yaml: true},
		},
	}
}

type extendedCodec struct {
	pretty bool
	yaml   bool
}

func (e *extendedCodec) Decode(data []byte, gvk *schema.GroupVersionKind, obj runtime.Object) (runtime.Object, *schema.GroupVersionKind, error) {
	if e.yaml {
		altered, err := yaml.YAMLToJSON(data)
		if err != nil {
			return nil, nil, err
		}
		data = altered
	}
	if obj == nil {
		metadata := &metav1.TypeMeta{}
		err := json.Unmarshal(data, metadata)
		if err != nil {
			return obj, gvk, err
		}
		log.V(7).Infoln("Detected metadata type for nil object, got", metadata.APIVersion, metadata.Kind)
		obj, err = setDefaultType(metadata)
		if err != nil {
			return obj, gvk, err
		}
	}
	err := json.Unmarshal(data, obj)
	if err != nil {
		return obj, gvk, err
	}
	return obj, gvk, nil
}

func (e *extendedCodec) Encode(obj runtime.Object, w io.Writer) error {
	setDefaultVersionKind(obj)
	if e.yaml {
		json, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		data, err := yaml.JSONToYAML(json)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		if err != nil {
			return err
		}
	}

	if e.pretty {
		data, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	return json.NewEncoder(w).Encode(obj)
}

// DecodeParameters converts the provided url.Values into an object of type From with the kind of into, and then
// converts that object to into (if necessary). Returns an error if the operation cannot be completed.
func (*extendedCodec) DecodeParameters(parameters url.Values, from schema.GroupVersion, into runtime.Object) error {
	if len(parameters) == 0 {
		return nil
	}
	return runtime.NewParameterCodec(api.Scheme).DecodeParameters(parameters, from, into)
}

// EncodeParameters converts the provided object into the to version, then converts that object to url.Values.
// Returns an error if conversion is not possible.
func (c *extendedCodec) EncodeParameters(obj runtime.Object, to schema.GroupVersion) (url.Values, error) {
	result := url.Values{}
	if obj == nil {
		return result, nil
	}
	return runtime.NewParameterCodec(api.Scheme).EncodeParameters(obj, to)
}

func setDefaultVersionKind(obj runtime.Object) {
	// Check the values can are In type Extended Ingress
	defaultGVK := schema.GroupVersionKind{
		Group:   sapi.V1alpha1SchemeGroupVersion.Group,
		Version: sapi.V1alpha1SchemeGroupVersion.Version,
	}

	fullyQualifiedKind := reflect.ValueOf(obj).Type().String()
	lastIndexOfDot := strings.LastIndex(fullyQualifiedKind, ".")
	if lastIndexOfDot > 0 {
		defaultGVK.Kind = fullyQualifiedKind[lastIndexOfDot+1:]
	}

	obj.GetObjectKind().SetGroupVersionKind(defaultGVK)
}

func setDefaultType(metadata *metav1.TypeMeta) (runtime.Object, error) {
	return api.Scheme.New(metadata.GroupVersionKind())
}
