package crdfuzz

import (
	"io/ioutil"
	"math/rand"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsinstall "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	crdv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	crdv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	structuralpruning "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/pruning"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	metafuzzer "k8s.io/apimachinery/pkg/apis/meta/fuzzer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	jsonserializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/versioning"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

const iters = 100

var (
	internalScheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(metav1.AddMetaToScheme(internalScheme))
	apiextensionsinstall.Install(internalScheme)
}

// SchemaFuzzTestForObject will run schema validation based pruning fuzz tests
// against a single given obj using the provided schema.
func SchemaFuzzTestForObject(t *testing.T, scheme *runtime.Scheme, obj runtime.Object, schema *structuralschema.Structural, fuzzingFuncs fuzzer.FuzzerFuncs) {
	codecFactory := serializer.NewCodecFactory(scheme)
	fuzzer := fuzzer.FuzzerFor(
		fuzzer.MergeFuzzerFuncs(metafuzzer.Funcs, fuzzingFuncs),
		rand.NewSource(rand.Int63()),
		codecFactory,
	)

	t.Logf("Running CRD schema pruning fuzz test for object %v", obj.GetObjectKind())
	for i := 0; i < iters; i++ {
		fuzzed := obj.DeepCopyObject()
		// fuzz *before* converting to unstructured, so we get typed fuzzing
		fuzzer.Fuzz(fuzzed)
		unstructuredFuzzed, err := runtime.DefaultUnstructuredConverter.ToUnstructured(fuzzed)
		if err != nil {
			t.Fatalf("Failed to convert type to `runtime.Unstructured`: %v", err)
			return
		}
		pruned, err := runtime.DefaultUnstructuredConverter.ToUnstructured(fuzzed)
		if err != nil {
			t.Fatalf("Failed to convert type to `runtime.Unstructured`: %v", err)
			return
		}
		structuralpruning.Prune(pruned, schema, true)
		if !cmp.Equal(unstructuredFuzzed, pruned) {
			t.Errorf("Failed fuzz test, difference: %v", cmp.Diff(unstructuredFuzzed, pruned))
		}
		t.Logf("Passed fuzz test iteration %d", i)
	}
}

// SchemaFuzzTestForInternalCRD will perform schema validation based pruning
// fuzz tests against all versions defined in a given CRD object.
func SchemaFuzzTestForInternalCRD(t *testing.T, scheme *runtime.Scheme, crd *apiextensions.CustomResourceDefinition, fuzzingFuncs fuzzer.FuzzerFuncs) {
	gk := schema.GroupKind{
		Group: crd.Spec.Group,
		Kind:  crd.Spec.Names.Kind,
	}

	var globalStructural *structuralschema.Structural
	if crd.Spec.Validation != nil {
		var err error
		globalStructural, err = structuralschema.NewStructural(crd.Spec.Validation.OpenAPIV3Schema)
		if err != nil {
			t.Errorf("Failed to construct structural schema: %v", err)
			return
		}
	}

	for _, vers := range crd.Spec.Versions {
		gvk := gk.WithVersion(vers.Name)
		t.Run(gvk.String(), func(t *testing.T) {
			obj, err := scheme.New(gvk)
			if err != nil {
				t.Errorf("Could not create Object with GroupVersionKind %v: %v", gvk, err)
				return
			}

			structural := globalStructural
			if structural == nil {
				if vers.Schema == nil {
					t.Errorf("GroupVersionKind %v has no schema defined, cannot run pruning tests", gvk)
					return
				}

				structural, err = structuralschema.NewStructural(vers.Schema.OpenAPIV3Schema)
				if err != nil {
					t.Errorf("Failed to construct structural schema: %v", err)
					return
				}
			}

			t.Logf("Using schema: %+v", structural)
			SchemaFuzzTestForObject(t, scheme, obj, structural, fuzzingFuncs)
		})
	}
}

// SchemaFuzzTestForCRDWithPath will perform schema validation based pruning
// fuzz tests against all versions defined in a file containing a single
// CustomResourceDefinition resource in any support CRD APIVersion (currently
// v1beta1 and v1)
func SchemaFuzzTestForCRDWithPath(t *testing.T, scheme *runtime.Scheme, path string, fuzzingFuncs fuzzer.FuzzerFuncs) {
	serializer := jsonserializer.NewSerializerWithOptions(jsonserializer.DefaultMetaFactory, internalScheme, internalScheme, jsonserializer.SerializerOptions{
		Yaml: true,
	})
	convertor := runtime.UnsafeObjectConvertor(internalScheme)
	codec := versioning.NewCodec(serializer, serializer, convertor, internalScheme, internalScheme, internalScheme, runtime.InternalGroupVersioner, runtime.InternalGroupVersioner, internalScheme.Name())

	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read CRD input file %q: %v", path, err)
		return
	}

	crd := &apiextensions.CustomResourceDefinition{}
	if _, _, err := codec.Decode(data, nil, crd); err != nil {
		t.Fatalf("Failed to decode CRD data: %v", err)
		return
	}

	SchemaFuzzTestForInternalCRD(t, scheme, crd, fuzzingFuncs)
}

// SchemaFuzzTestForV1beta1CRD will perform schema validation based pruning
// fuzz tests against all versions defined in a given v1beta1 CRD object.
func SchemaFuzzTestForV1beta1CRD(t *testing.T, scheme *runtime.Scheme, crd *crdv1beta1.CustomResourceDefinition, fuzzingFuncs fuzzer.FuzzerFuncs) {
	var internalCRD apiextensions.CustomResourceDefinition
	err := crdv1beta1.Convert_v1beta1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(crd, &internalCRD, nil)
	if err != nil {
		t.Fatalf("Failed to convert v1beta1 CRD to internal CRD: %v", err)
		return
	}
	SchemaFuzzTestForInternalCRD(t, scheme, &internalCRD, fuzzingFuncs)
}

// SchemaFuzzTestForV1CRD will perform schema validation based pruning
// fuzz tests against all versions defined in a given v1 CRD object.
func SchemaFuzzTestForV1CRD(t *testing.T, scheme *runtime.Scheme, crd *crdv1.CustomResourceDefinition, fuzzingFuncs fuzzer.FuzzerFuncs) {
	var internalCRD apiextensions.CustomResourceDefinition
	err := crdv1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(crd, &internalCRD, nil)
	if err != nil {
		t.Fatalf("Failed to convert v1 CRD to internal CRD: %v", err)
		return
	}
	SchemaFuzzTestForInternalCRD(t, scheme, &internalCRD, fuzzingFuncs)
}
