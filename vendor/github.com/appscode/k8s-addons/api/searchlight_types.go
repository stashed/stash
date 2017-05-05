package api

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
)

// Alert types for appscode.
type Alert struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: http://releases.k8s.io/release-1.2/docs/devel/api-conventions.md#metadata
	api.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the desired state of the Alert.
	// More info: http://releases.k8s.io/release-1.2/docs/devel/api-conventions.md#spec-and-status
	Spec AlertSpec `json:"spec,omitempty"`

	// Status is the current state of the Alert.
	// More info: http://releases.k8s.io/release-1.2/docs/devel/api-conventions.md#spec-and-status
	Status AlertStatus `json:"status,omitempty"`
}

type AlertPhase string

const (
	// used for Alert that are currently creating
	AlertPhaseCreating AlertPhase = "Creating"
	// used for Alert that are created
	AlertPhaseCreated AlertPhase = "Created"
	// used for Alert that are currently deleting
	AlertPhaseDeleting AlertPhase = "Deleting"
	// used for Alert that are Failed
	AlertPhaseFailed AlertPhase = "Failed"
)

type AlertStatus struct {
	CreationTime *unversioned.Time `json:"creationTime,omitempty"`
	UpdateTime   *unversioned.Time `json:"updateTime,omitempty"`
	Phase        AlertPhase        `json:"phase,omitempty"`
	Reason       string            `json:"reason,omitempty"`
}

// AlertList is a collection of Alert.
type AlertList struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: http://releases.k8s.io/release-1.2/docs/devel/api-conventions.md#metadata
	unversioned.ListMeta `json:"metadata,omitempty"`

	// Items is the list of Alert.
	Items []Alert `json:"items"`
}

// AlertSpec describes the Alert the user wishes to create.
type AlertSpec struct {
	// IcingaParam contains parameters for Icinga config
	IcingaParam *IcingaParam `json:"icingaParam,omitempty"`

	// Icinga CheckCommand name
	CheckCommand string `json:"checkCommand,omitempty"`

	// NotifierParams contains information to send notifications for Incident
	// State, UserUid, Method
	NotifierParams []NotifierParam `json:"notifierParams,omitempty"`

	// Vars contains Icinga Service variables to be used in CheckCommand
	Vars map[string]interface{} `json:"vars,omitempty"`
}

type IcingaParam struct {
	// How frequently Icinga Service will be checked
	CheckIntervalSec int64 `json:"checkIntervalSec,omitempty"`

	// How frequently notifications will be send
	AlertIntervalSec int64 `json:"alertIntervalSec,omitempty"`
}

type NotifierParam struct {
	// For which state notification will be sent
	State string `json:"state,omitempty"`

	// To whom notification will be sent
	UserUid string `json:"userUid,omitempty"`

	// How this notification will be sent
	Method string `json:"method,omitempty"`
}
