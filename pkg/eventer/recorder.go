/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package eventer

import (
	"context"
	"fmt"
	"time"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"
	"k8s.io/klog/v2"
)

const (
	EventReasonAdmissionWebhookNotActivated = "AdmissionWebhookNotActivated"

	// ====================== Event Sources ===================================
	EventSourceBackupConfigurationController = "BackupConfiguration Controller"
	EventSourceBackupSessionController       = "BackupSession Controller"
	EventSourceRestoreSessionController      = "RestoreSession Controller"
	EventSourceWorkloadController            = "Workload Controller"
	EventSourceBackupSidecar                 = "Backup Sidecar"
	EventSourceRestoreInitContainer          = "Restore Init-Container"
	EventSourceBackupTriggeringCronJob       = "Backup Triggering CronJob"
	EventSourceStatusUpdater                 = "Status Updater"

	// ======================= Event Reasons ========================
	// BackupConfiguration Events
	EventReasonCronJobCreationFailed = "CronJob Creation Failed"
	// BackupSession Events
	EventReasonHostBackupSucceded = "Host Backup Succeeded"
	EventReasonHostBackupFailed   = "Host Backup Failed"
	// Restore Events
	EventReasonHostRestoreSucceeded = "Host Restore Succeeded"
	EventReasonHostRestoreFailed    = "Host Restore Failed"

	// Sidecar Events
	EventReasonSidecarInjectionFailed               = "Sidecar Injection Failed"
	EventReasonSidecarInjectionSucceeded            = "Sidecar Injection Succeeded"
	EventReasonSidecarDeletionFailed                = "Sidecar Deletion Failed"
	EventReasonSidecarDeletionSucceeded             = "Sidecar Deletion Succeeded"
	EventReasonFailedToStartBackupSessionController = "Failed To Start BackupSession Controller"
	EventReasonBackupSessionControllerStarted       = "BackupSession Controller Started"
	// Init-container Events
	EventReasonInitContainerInjectionFailed    = "Init-Container Injection Failed"
	EventReasonInitContainerInjectionSucceeded = "Init-Container Injection Succeeded"
	EventReasonInitContainerDeletionSucceeded  = "Init-Container Deletion Succeeded"

	EventReasonWorkloadControllerTriggeringFailed = "Failed To Trigger Workload Controller"
)

func NewEventRecorder(client kubernetes.Interface, component string) record.EventRecorder {
	// Event Broadcaster
	broadcaster := record.NewBroadcaster()
	broadcaster.StartEventWatcher(
		func(event *core.Event) {
			if _, err := client.CoreV1().Events(event.Namespace).Create(context.TODO(), event, metav1.CreateOptions{}); err != nil {
				klog.Errorln(err)
			}
		},
	)
	// Event Recorder
	return broadcaster.NewRecorder(scheme.Scheme, core.EventSource{Component: component})
}

func CreateEvent(client kubernetes.Interface, component string, obj runtime.Object, eventType, reason, message string) (*core.Event, error) {
	ref, err := reference.GetReference(scheme.Scheme, obj)
	if err != nil {
		return nil, err
	}

	t := metav1.Time{Time: time.Now()}

	return client.CoreV1().Events(ref.Namespace).Create(context.TODO(), &core.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v.%x", ref.Name, t.UnixNano()),
			Namespace: ref.Namespace,
		},
		InvolvedObject: *ref,
		Reason:         reason,
		Message:        message,
		FirstTimestamp: t,
		LastTimestamp:  t,
		Count:          1,
		Type:           eventType,
		Source:         core.EventSource{Component: component},
	}, metav1.CreateOptions{})
}

func CreateEventWithLog(client kubernetes.Interface, component string, obj runtime.Object, eventType, reason, message string) {
	event, err := CreateEvent(client, component, obj, eventType, reason, message)
	if err != nil {
		klog.Errorln("Failed to write event, reason: ", err)
		return
	}
	klog.Infoln("Event created: ", event.Name)
}
