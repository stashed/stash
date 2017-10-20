package kutil

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	apiv1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

const (
	RetryInterval = 50 * time.Millisecond
	RetryTimeout  = 2 * time.Second
)

func Namespace() string {
	if ns := os.Getenv("KUBE_NAMESPACE"); ns != "" {
		return ns
	}
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}
	return apiv1.NamespaceDefault
}

func IsPreferredAPIResource(c kubernetes.Interface, groupVersion, kind string) bool {
	if resourceList, err := c.Discovery().ServerPreferredResources(); err == nil {
		for _, resources := range resourceList {
			if resources.GroupVersion != groupVersion {
				continue
			}
			for _, resource := range resources.APIResources {
				if resources.GroupVersion == groupVersion && resource.Kind == kind {
					return true
				}
			}
		}
	}
	return false
}

func CheckAPIVersion(c kubernetes.Interface, constraint string) (bool, error) {
	info, err := c.Discovery().ServerVersion()
	if err != nil {
		return false, err
	}
	cond, err := version.NewConstraint(constraint)
	if err != nil {
		return false, err
	}
	v, err := version.NewVersion(info.GitVersion)
	if err != nil {
		return false, err
	}
	return cond.Check(v.ToMutator().ResetPrerelease().ResetMetadata().Done()), nil
}

func WaitForCRDReady(restClient rest.Interface, crds []*apiextensions.CustomResourceDefinition) error {
	err := wait.Poll(3*time.Second, 5*time.Minute, func() (bool, error) {
		for _, crd := range crds {
			res := restClient.Get().AbsPath("apis", crd.Spec.Group, crd.Spec.Version, crd.Spec.Names.Plural).Do()
			err := res.Error()
			if err != nil {
				// RESTClient returns *apierrors.StatusError for any status codes < 200 or > 206
				// and http.Client.Do errors are returned directly.
				if se, ok := err.(*kerr.StatusError); ok {
					if se.Status().Code == http.StatusNotFound {
						return false, nil
					}
				}
				return false, err
			}

			var statusCode int
			res.StatusCode(&statusCode)
			if statusCode != http.StatusOK {
				return false, fmt.Errorf("invalid status code: %d", statusCode)
			}
		}

		return true, nil
	})

	return errors.Wrap(err, fmt.Sprintf("timed out waiting for CRD"))
}

func DeleteInBackground() *metav1.DeleteOptions {
	policy := metav1.DeletePropagationBackground
	return &metav1.DeleteOptions{PropagationPolicy: &policy}
}

func DeleteInForeground() *metav1.DeleteOptions {
	policy := metav1.DeletePropagationForeground
	return &metav1.DeleteOptions{PropagationPolicy: &policy}
}

func GetKind(v interface{}) string {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return val.Type().Name()
}

func GetObjectReference(v interface{}, gv schema.GroupVersion) *apiv1.ObjectReference {
	m, err := meta.Accessor(v)
	if err != nil {
		return &apiv1.ObjectReference{}
	}
	return &apiv1.ObjectReference{
		APIVersion:      gv.String(),
		Kind:            GetKind(v),
		Namespace:       m.GetNamespace(),
		Name:            m.GetName(),
		UID:             m.GetUID(),
		ResourceVersion: m.GetResourceVersion(),
	}
}

// MarshalToYAML marshals an object into yaml.
func MarshalToYAML(obj runtime.Object, gv schema.GroupVersion) ([]byte, error) {
	mediaType := "application/yaml"
	info, ok := runtime.SerializerInfoForMediaType(clientsetscheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return []byte{}, fmt.Errorf("unsupported media type %q", mediaType)
	}

	encoder := clientsetscheme.Codecs.EncoderForVersion(info.Serializer, gv)
	return runtime.Encode(encoder, obj)
}

// UnmarshalToYAML unmarshals an object into yaml.
func UnmarshalToYAML(data []byte, gv schema.GroupVersion) (runtime.Object, error) {
	mediaType := "application/yaml"
	info, ok := runtime.SerializerInfoForMediaType(clientsetscheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return nil, fmt.Errorf("unsupported media type %q", mediaType)
	}

	decoder := clientsetscheme.Codecs.DecoderToVersion(info.Serializer, gv)
	return runtime.Decode(decoder, data)
}

// MarshalToJson marshals an object into json.
func MarshalToJson(obj runtime.Object, gv schema.GroupVersion) ([]byte, error) {
	mediaType := "application/json"
	info, ok := runtime.SerializerInfoForMediaType(clientsetscheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return []byte{}, fmt.Errorf("unsupported media type %q", mediaType)
	}

	encoder := clientsetscheme.Codecs.EncoderForVersion(info.Serializer, gv)
	return runtime.Encode(encoder, obj)
}

// UnmarshalToJSON unmarshals an object into json.
func UnmarshalToJSON(data []byte, gv schema.GroupVersion) (runtime.Object, error) {
	mediaType := "application/json"
	info, ok := runtime.SerializerInfoForMediaType(clientsetscheme.Codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return nil, fmt.Errorf("unsupported media type %q", mediaType)
	}

	decoder := clientsetscheme.Codecs.DecoderToVersion(info.Serializer, gv)
	return runtime.Decode(decoder, data)
}
