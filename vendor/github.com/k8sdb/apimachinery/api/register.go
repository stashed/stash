package api

import (
	"k8s.io/kubernetes/pkg/api"
	schema "k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/runtime"
)

// GroupName is the group name use in this package
const GroupName = "kubedb.com"

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: runtime.APIVersionInternal}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns back a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
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

		&api.ListOptions{},
	)
	return nil
}

func (obj *Snapshot) GetObjectKind() schema.ObjectKind     { return &obj.TypeMeta }
func (obj *SnapshotList) GetObjectKind() schema.ObjectKind { return &obj.TypeMeta }

func (obj *DormantDatabase) GetObjectKind() schema.ObjectKind     { return &obj.TypeMeta }
func (obj *DormantDatabaseList) GetObjectKind() schema.ObjectKind { return &obj.TypeMeta }

func (obj *Elastic) GetObjectKind() schema.ObjectKind     { return &obj.TypeMeta }
func (obj *ElasticList) GetObjectKind() schema.ObjectKind { return &obj.TypeMeta }

func (obj *Postgres) GetObjectKind() schema.ObjectKind     { return &obj.TypeMeta }
func (obj *PostgresList) GetObjectKind() schema.ObjectKind { return &obj.TypeMeta }
