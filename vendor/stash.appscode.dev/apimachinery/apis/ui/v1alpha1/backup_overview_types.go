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
	api "stash.appscode.dev/apimachinery/apis/stash/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceKindBackupOverview = "BackupOverview"
	ResourceBackupOverview     = "backupoverview"
	ResourceBackupOverviews    = "backupoverviews"
)

// +kubebuilder:validation:Enum=Active;Paused
type BackupStatus string

const (
	BackupStatusActive = "Active"
	BackupStatusPaused = "Paused"
)

// BackupOverviewSpec defines the desired state of BackupOverview
type BackupOverviewSpec struct {
	Schedule           string       `json:"schedule,omitempty"`
	Status             BackupStatus `json:"status,omitempty"`
	LastBackupTime     *metav1.Time `json:"lastBackupTime,omitempty"`
	UpcomingBackupTime *metav1.Time `json:"upcomingBackupTime,omitempty"`
	Repository         string       `json:"repository,omitempty"`
	DataSize           string       `json:"dataSize,omitempty"`
	NumberOfSnapshots  int64        `json:"numberOfSnapshots,omitempty"`
	DataIntegrity      bool         `json:"dataIntegrity,omitempty"`
}

// BackupOverview is the Schema for the BackupOverviews API

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BackupOverview struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupOverviewSpec            `json:"spec,omitempty"`
	Status api.BackupConfigurationStatus `json:"status,omitempty"`
}

// BackupOverviewList contains a list of BackupOverview

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BackupOverviewList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupOverview `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupOverview{}, &BackupOverviewList{})
}
