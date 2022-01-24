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

	// UsagePolicy specifies a policy of how this Repository will be used. For example, you can use `allowedNamespaces`
	// policy to restrict the usage of this Repository to particular namespaces.
	// This field is optional. If you don't provide the usagePolicy, then it can be used only from the current namespace.
	// +optional
	UsagePolicy *UsagePolicy `json:"usagePolicy,omitempty" protobuf:"bytes,3,opt,name=usagePolicy"`
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
}

// UsagePolicy specifies a policy that restrict the usage of a resource across namespaces.
type UsagePolicy struct {
	// AllowedNamespaces specifies which namespaces are allowed to use the resource
	// +optional
	AllowedNamespaces AllowedNamespaces `json:"allowedNamespaces,omitempty" protobuf:"bytes,1,opt,name=allowedNamespaces"`
}

// AllowedNamespaces indicate which namespaces the resource should be selected from.
type AllowedNamespaces struct {
	// From indicates how to select the namespaces that are allowed to use this resource.
	// Possible values are:
	// * All: All namespaces can use this resource.
	// * Selector: Namespaces that matches the selector can use this resource.
	// * Same: Only current namespace can use the resource.
	//
	// +optional
	// +kubebuilder:default=Same
	From *FromNamespaces `json:"from,omitempty" protobuf:"bytes,1,opt,name=from,casttype=FromNamespaces"`

	// Selector must be specified when From is set to "Selector". In that case,
	// only the selected namespaces are allowed to use this resource.
	// This field is ignored for other values of "From".
	//
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty" protobuf:"bytes,2,opt,name=selector"`
}

// FromNamespaces specifies namespace from which namespaces are allowed to use the resource.
//
// +kubebuilder:validation:Enum=All;Selector;Same
type FromNamespaces string

const (
	// NamespacesFromAll specifies that all namespaces can use the resource.
	NamespacesFromAll FromNamespaces = "All"

	// NamespacesFromSelector specifies that only the namespace that matches the selector can use the resource.
	NamespacesFromSelector FromNamespaces = "Selector"

	// NamespacesFromSame specifies that only the current namespace can use the resource.
	NamespacesFromSame FromNamespaces = "Same"
)

// +kubebuilder:validation:Enum=--keep-last;--keep-hourly;--keep-daily;--keep-weekly;--keep-monthly;--keep-yearly;--keep-tag
type RetentionStrategy string

const (
	KeepLast    RetentionStrategy = "--keep-last"
	KeepHourly  RetentionStrategy = "--keep-hourly"
	KeepDaily   RetentionStrategy = "--keep-daily"
	KeepWeekly  RetentionStrategy = "--keep-weekly"
	KeepMonthly RetentionStrategy = "--keep-monthly"
	KeepYearly  RetentionStrategy = "--keep-yearly"
	KeepTag     RetentionStrategy = "--keep-tag"
)

type RetentionPolicy struct {
	Name        string   `json:"name" protobuf:"bytes,1,opt,name=name"`
	KeepLast    int64    `json:"keepLast,omitempty" protobuf:"varint,2,opt,name=keepLast"`
	KeepHourly  int64    `json:"keepHourly,omitempty" protobuf:"varint,3,opt,name=keepHourly"`
	KeepDaily   int64    `json:"keepDaily,omitempty" protobuf:"varint,4,opt,name=keepDaily"`
	KeepWeekly  int64    `json:"keepWeekly,omitempty" protobuf:"varint,5,opt,name=keepWeekly"`
	KeepMonthly int64    `json:"keepMonthly,omitempty" protobuf:"varint,6,opt,name=keepMonthly"`
	KeepYearly  int64    `json:"keepYearly,omitempty" protobuf:"varint,7,opt,name=keepYearly"`
	KeepTags    []string `json:"keepTags,omitempty" protobuf:"bytes,8,rep,name=keepTags"`
	Prune       bool     `json:"prune" protobuf:"varint,9,opt,name=prune"`
	DryRun      bool     `json:"dryRun,omitempty" protobuf:"varint,10,opt,name=dryRun"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Repository `json:"items,omitempty" protobuf:"bytes,2,rep,name=items"`
}
