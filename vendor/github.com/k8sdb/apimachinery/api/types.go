package api

import "k8s.io/kubernetes/pkg/api"

// StorageSpec defines storage provisioning
type StorageSpec struct {
	// Name of the StorageClass to use when requesting storage provisioning.
	Class string `json:"class"`
	// Persistent Volume Claim
	api.PersistentVolumeClaimSpec `json:",inline,omitempty"`
}

type InitSpec struct {
	ScriptSource   *ScriptSourceSpec   `json:"scriptSource,omitempty"`
	SnapshotSource *SnapshotSourceSpec `json:"snapshotSource,omitempty"`
}

type ScriptSourceSpec struct {
	ScriptPath       string `json:"scriptPath,omitempty"`
	api.VolumeSource `json:",inline,omitempty"`
}

type SnapshotSourceSpec struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

type BackupScheduleSpec struct {
	CronExpression      string `json:"cronExpression,omitempty"`
	SnapshotStorageSpec `json:",inline,omitempty"`
}

type SnapshotStorageSpec struct {
	// Snapshot storage secret
	StorageSecret *api.SecretVolumeSource `json:"storageSecret,omitempty"`
	// Cloud bucket name
	BucketName string `json:"bucketName,omitempty"`
}

type MonitorSpec struct {
	Prometheus *PrometheusSpec `json:"prometheus,omitempty"`
}

type PrometheusSpec struct {
	// Namespace of Prometheus. Service monitors will be created in this namespace.
	Namespace string `json:"namespace,omitempty"`
	// Labels are key value pairs that is used to select Prometheus instance via ServiceMonitor labels.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Interval at which metrics should be scraped
	Interval string `json:"interval,omitempty"`

	// Parameters are key value pairs that are passed as flags to exporters.
	// Parameters map[string]string `json:"parameters,omitempty"`
}

type DatabasePhase string

const (
	// used for Databases that are currently running
	DatabasePhaseRunning DatabasePhase = "Running"
	// used for Databases that are currently creating
	DatabasePhaseCreating DatabasePhase = "Creating"
	// used for Databases that are currently initializing
	DatabasePhaseInitializing DatabasePhase = "Initializing"
	// used for Databases that are Failed
	DatabasePhaseFailed DatabasePhase = "Failed"
)
