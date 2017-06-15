package api

import (
apiv1 "k8s.io/client-go/pkg/api/v1"
metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
"k8s.io/apimachinery/pkg/runtime"
)

// GroupName is the group name use in this package
const GroupName = "backup.appscode.com"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = metav1.GroupVersion{Group: GroupName, Version: runtime.APIVersionInternal}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) metav1.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns back a Group qualified GroupResource
func Resource(resource string) metav1.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to apiv1.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,

		&Restik{},
		&RestikList{},

		&apiv1.ListOptions{},
	)
	return nil
}

func (obj *Restik) GetObjectKind() metav1.ObjectKind     { return &obj.TypeMeta }
func (obj *RestikList) GetObjectKind() metav1.ObjectKind { return &obj.TypeMeta }
