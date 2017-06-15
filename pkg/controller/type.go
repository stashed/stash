package controller

import (
	"time"

	rapi "github.com/appscode/restik/api"
	tcs "github.com/appscode/restik/client/clientset"
	"gopkg.in/robfig/cron.v2"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

const (
	BackupConfig          = "backup.appscode.com/config"
	ContainerName         = "restic-sidecar"
	RestikNamespace       = "RESTIK_NAMESPACE"
	RestikResourceName    = "RESTIK_RESOURCE_NAME"
	RESTIC_PASSWORD       = "RESTIC_PASSWORD"
	ReplicationController = "ReplicationController"
	ReplicaSet            = "ReplicaSet"
	Deployment            = "Deployment"
	DaemonSet             = "DaemonSet"
	StatefulSet           = "StatefulSet"
	Password              = "password"
	ImageAnnotation       = "backup.appscode.com/image"
	Force                 = "force"
)

const (
	EventReasonInvalidCronExpression         = "InvalidCronExpression"
	EventReasonSuccessfulCronExpressionReset = "SuccessfulCronExpressionReset"
	EventReasonSuccessfulBackup              = "SuccessfulBackup"
	EventReasonFailedToBackup                = "FailedBackup"
	EventReasonFailedToRetention             = "FailedRetention"
	EventReasonFailedToUpdate                = "FailedUpdateBackup"
	EventReasonFailedCronJob                 = "FailedCronJob"
)

type Controller struct {
	ExtClientset tcs.ExtensionInterface
	Clientset    clientset.Interface
	// sync time to sync the list.
	SyncPeriod time.Duration
	// image of sidecar container
	Image string
}

type cronController struct {
	extClientset  tcs.ExtensionInterface
	clientset     clientset.Interface
	tprName       string
	namespace     string
	crons         *cron.Cron
	restik        *rapi.Restik
	eventRecorder record.EventRecorder
}
