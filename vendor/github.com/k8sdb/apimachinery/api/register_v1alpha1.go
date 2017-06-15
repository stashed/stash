package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion is group version used to register these objects
var V1alpha1SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}

var (
	V1alpha1SchemeBuilder = runtime.NewSchemeBuilder(v1addKnownTypes)
	V1alpha1AddToScheme   = V1alpha1SchemeBuilder.AddToScheme
)

// Adds the list of known types to api.Scheme.
func v1addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(V1alpha1SchemeGroupVersion,
		// Snapshot
		&Snapshot{},
		&SnapshotList{},
		// DormantDatabase
		&DormantDatabase{},
		&DormantDatabaseList{},
		// kubedb Elastic
		&Elastic{},
		&ElasticList{},
		// kubedb Postgres
		&Postgres{},
		&PostgresList{},

		&metav1.ListOptions{},
	)
	metav1.AddToGroupVersion(scheme, V1alpha1SchemeGroupVersion)
	return nil
}
