package controller

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
	EventReasonCronExpressionFailed = "Failed"
	EventReasonBackupSuccess        = "Success"
	EventReasonBackupFailed         = "Failed"
)
