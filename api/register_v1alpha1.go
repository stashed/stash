package api

import (
metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/api/v1"
"k8s.io/apimachinery/pkg/runtime"
	versionedwatch "k8s.io/kubernetes/pkg/watch/versioned"
)

// SchemeGroupVersion is group version used to register these objects
var V1alpha1SchemeGroupVersion = metav1.GroupVersion{Group: GroupName, Version: "v1alpha1"}

var (
	V1alpha1SchemeBuilder = runtime.NewSchemeBuilder(v1addKnownTypes, addConversionFuncs)
	V1alpha1AddToScheme   = V1alpha1SchemeBuilder.AddToScheme
)

// Adds the list of known types to api.Scheme.
func v1addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(V1alpha1SchemeGroupVersion,

		&Restik{},
		&RestikList{},

		&v1.ListOptions{},
	)
	versionedwatch.AddToGroupVersion(scheme, V1alpha1SchemeGroupVersion)
	return nil
}
