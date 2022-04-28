/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"stash.appscode.dev/apimachinery/apis/stash/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

const (
	ResourceKindBackupBlueprint     = "BackupBlueprint"
	ResourcePluralBackupBlueprint   = "backupblueprints"
	ResourceSingularBackupBlueprint = "backupblueprint"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=backupblueprints,singular=backupblueprint,scope=Cluster,shortName=bb,categories={stash,appscode}
// +kubebuilder:printcolumn:name="Task",type="string",JSONPath=".spec.task.name"
// +kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=".spec.schedule"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type BackupBlueprint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BackupBlueprintSpec `json:"spec,omitempty"`
}

type BackupBlueprintSpec struct {
	// RepositorySpec is used to create Repository crd for respective workload
	v1alpha1.RepositorySpec `json:",inline"`
	// BackupNamespace specifies the namespace where the backup resources (i.e. BackupConfiguration, BackupSession, Job, Repository etc.) will be created.
	// If you don't provide this field, then the backup resources will be created in the target namespace.
	// +optional
	BackupNamespace string `json:"backupNamespace,omitempty"`
	// RepoNamespace lets you specify the namespace for the Repositories. If this field is not specified, Stash will create the Repository
	// in the namespace pointed by the backupNamespace field. If neither of the backupNamespace and repoNamespace is specified,
	// Stash will create the Repository in the target namespace.
	// +optional
	RepoNamespace string `json:"repoNamespace,omitempty"`
	// Schedule specifies the default schedule for backup.
	// You can overwrite this schedule for a particular target using 'stash.appscode.com/schedule' annotation.
	Schedule string `json:"schedule,omitempty"`
	// Task specify the Task crd that specifies steps for backup process
	// +optional
	Task TaskRef `json:"task,omitempty"`
	// RetentionPolicy indicates the policy to follow to clean old backup snapshots
	RetentionPolicy v1alpha1.RetentionPolicy `json:"retentionPolicy"`
	// RuntimeSettings allow to specify Resources, NodeSelector, Affinity, Toleration, ReadinessProbe etc.
	// +optional
	RuntimeSettings ofst.RuntimeSettings `json:"runtimeSettings,omitempty"`
	// Temp directory configuration for functions/sidecar
	// An `EmptyDir` will always be mounted at /tmp with this settings
	// +optional
	TempDir EmptyDirSettings `json:"tempDir,omitempty"`
	// InterimVolumeTemplate specifies a template for a volume to hold targeted data temporarily
	// before uploading to backend or inserting into target. It is only usable for job model.
	// Don't specify it in sidecar model.
	// +optional
	InterimVolumeTemplate *ofst.PersistentVolumeClaim `json:"interimVolumeTemplate,omitempty"`
	// BackupHistoryLimit specifies the number of BackupSession and it's associate resources to keep.
	// This is helpful for debugging purpose.
	// Default: 1
	// +optional
	BackupHistoryLimit *int32 `json:"backupHistoryLimit,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BackupBlueprintList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupBlueprint `json:"items,omitempty"`
}
