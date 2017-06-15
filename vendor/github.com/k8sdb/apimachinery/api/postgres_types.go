package api

import (
	"github.com/appscode/go/encoding/json/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	ResourceCodePostgres = "pg"
	ResourceKindPostgres = "Postgres"
	ResourceNamePostgres = "postgres"
	ResourceTypePostgres = "postgreses"
)

// Postgres defines a Postgres database.
type Postgres struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PostgresSpec   `json:"spec,omitempty"`
	Status            PostgresStatus `json:"status,omitempty"`
}

type PostgresSpec struct {
	// Version of Postgres to be deployed.
	Version types.StrYo `json:"version,omitempty"`
	// Storage spec to specify how storage shall be used.
	Storage *StorageSpec `json:"storage,omitempty"`
	// Database authentication secret
	DatabaseSecret *apiv1.SecretVolumeSource `json:"databaseSecret,omitempty"`
	// NodeSelector is a selector which must be true for the pod to fit on a node
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Init is used to initialize database
	// +optional
	Init *InitSpec `json:"init,omitempty"`
	// BackupSchedule spec to specify how database backup will be taken
	// +optional
	BackupSchedule *BackupScheduleSpec `json:"backupSchedule,omitempty"`
	// If DoNotPause is true, controller will prevent to delete this Postgres object.
	// Controller will create same Postgres object and ignore other process.
	// +optional
	DoNotPause bool `json:"doNotPause,omitempty"`
	// Monitor is used monitor database instance
	// +optional
	Monitor *MonitorSpec `json:"monitor,omitempty"`
}

type PostgresStatus struct {
	CreationTime *metav1.Time  `json:"creationTime,omitempty"`
	Phase        DatabasePhase `json:"phase,omitempty"`
	Reason       string        `json:"reason,omitempty"`
}

type PostgresList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// Items is a list of Postgres TPR objects
	Items []Postgres `json:"items,omitempty"`
}
