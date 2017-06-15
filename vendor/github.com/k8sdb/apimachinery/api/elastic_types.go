package api

import (
	"github.com/appscode/go/encoding/json/types"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

const (
	ResourceCodeElastic = "es"
	ResourceKindElastic = "Elastic"
	ResourceNameElastic = "elastic"
	ResourceTypeElastic = "elastics"
)

// Elastic defines a Elasticsearch database.
type Elastic struct {
	unversioned.TypeMeta `json:",inline,omitempty"`
	api.ObjectMeta       `json:"metadata,omitempty"`
	Spec                 ElasticSpec   `json:"spec,omitempty"`
	Status               ElasticStatus `json:"status,omitempty"`
}

type ElasticSpec struct {
	// Version of Elasticsearch to be deployed.
	Version types.StrYo `json:"version,omitempty"`
	// Number of instances to deploy for a Elasticsearch database.
	Replicas int32 `json:"replicas,omitempty"`
	// Storage spec to specify how storage shall be used.
	Storage *StorageSpec `json:"storage,omitempty"`
	// NodeSelector is a selector which must be true for the pod to fit on a node
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Init is used to initialize database
	// +optional
	Init *InitSpec `json:"init,omitempty"`
	// BackupSchedule spec to specify how database backup will be taken
	// +optional
	BackupSchedule *BackupScheduleSpec `json:"backupSchedule,omitempty"`
	// If DoNotPause is true, controller will prevent to delete this Elastic object.
	// Controller will create same Elastic object and ignore other process.
	// +optional
	DoNotPause bool `json:"doNotPause,omitempty"`
	// Monitor is used monitor database instance
	// +optional
	Monitor *MonitorSpec `json:"monitor,omitempty"`
}

type ElasticStatus struct {
	CreationTime *unversioned.Time `json:"creationTime,omitempty"`
	Phase        DatabasePhase     `json:"phase,omitempty"`
	Reason       string            `json:"reason,omitempty"`
}

type ElasticList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	// Items is a list of Elastic TPR objects
	Items []Elastic `json:"items,omitempty"`
}
