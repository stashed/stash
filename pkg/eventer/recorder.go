package eventer

import (
	"github.com/appscode/go/log"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
)

const (
	EventReasonInvalidRestic                 = "InvalidRestic"
	EventReasonInvalidRecovery               = "InvalidRecovery"
	EventReasonInvalidCronExpression         = "InvalidCronExpression"
	EventReasonSuccessfulCronExpressionReset = "SuccessfulCronExpressionReset"
	EventReasonSuccessfulBackup              = "SuccessfulBackup"
	EventReasonFailedToBackup                = "FailedBackup"
	EventReasonFailedToRetention             = "FailedRetention"
	EventReasonFailedToUpdate                = "FailedUpdateBackup"
	EventReasonFailedCronJob                 = "FailedCronJob"
)

func NewEventRecorder(client kubernetes.Interface, component string) record.EventRecorder {
	// Event Broadcaster
	broadcaster := record.NewBroadcaster()
	broadcaster.StartEventWatcher(
		func(event *apiv1.Event) {
			if _, err := client.CoreV1().Events(event.Namespace).Create(event); err != nil {
				log.Errorln(err)
			}
		},
	)
	// Event Recorder
	return broadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{Component: component})
}
