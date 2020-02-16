/*
Copyright The Stash Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	store "kmodules.xyz/objectstore-api/api/v1"
)

const (
	ResourceKindRepository     = "Repository"
	ResourcePluralRepository   = "repositories"
	ResourceSingularRepository = "repository"
)

// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=repositories,singular=repository,shortName=repo,categories={stash,appscode}
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Integrity",type="boolean",JSONPath=".status.integrity"
// +kubebuilder:printcolumn:name="Size",type="string",JSONPath=".status.totalSize"
// +kubebuilder:printcolumn:name="Snapshot-Count",type="integer",JSONPath=".status.snapshotCount"
// +kubebuilder:printcolumn:name="Last-Successful-Backup",type="date",format="date-time",JSONPath=".status.lastBackupTime"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              RepositorySpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status            RepositoryStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type RepositorySpec struct {
	// Backend specify the storage where backed up snapshot will be stored
	Backend store.Backend `json:"backend,omitempty" protobuf:"bytes,1,opt,name=backend"`
	// If true, delete respective restic repository
	// +optional
	WipeOut bool `json:"wipeOut,omitempty" protobuf:"varint,2,opt,name=wipeOut"`
}

type RepositoryStatus struct {
	// ObservedGeneration is the most recent generation observed for this Repository. It corresponds to the
	// Repository's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,1,opt,name=observedGeneration"`
	// FirstBackupTime indicates the timestamp when the first backup was taken
	FirstBackupTime *metav1.Time `json:"firstBackupTime,omitempty" protobuf:"bytes,2,opt,name=firstBackupTime"`
	// LastBackupTime indicates the timestamp when the latest backup was taken
	LastBackupTime *metav1.Time `json:"lastBackupTime,omitempty" protobuf:"bytes,3,opt,name=lastBackupTime"`
	// Integrity shows result of repository integrity check after last backup
	Integrity *bool `json:"integrity,omitempty" protobuf:"varint,4,opt,name=integrity"`
	// TotalSize show size of repository after last backup
	TotalSize string `json:"totalSize,omitempty" protobuf:"bytes,11,opt,name=totalSize"`
	// SnapshotCount shows number of snapshots stored in the repository
	SnapshotCount int64 `json:"snapshotCount,omitempty" protobuf:"varint,6,opt,name=snapshotCount"`
	// SnapshotsRemovedOnLastCleanup shows number of old snapshots cleaned up according to retention policy on last backup session
	SnapshotsRemovedOnLastCleanup int64 `json:"snapshotsRemovedOnLastCleanup,omitempty" protobuf:"varint,7,opt,name=snapshotsRemovedOnLastCleanup"`

	// Deprecated
	LastSuccessfulBackupTime *metav1.Time `json:"lastSuccessfulBackupTime,omitempty" protobuf:"bytes,8,opt,name=lastSuccessfulBackupTime"`
	// Deprecated
	LastBackupDuration string `json:"lastBackupDuration,omitempty" protobuf:"bytes,9,opt,name=lastBackupDuration"`
	// Deprecated
	BackupCount int64 `json:"backupCount,omitempty" protobuf:"varint,10,opt,name=backupCount"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Repository `json:"items,omitempty" protobuf:"bytes,2,rep,name=items"`
}
